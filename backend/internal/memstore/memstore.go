// Package memstore is an in-memory implementation of the expense and user
// repositories. It mirrors the PostgreSQL semantics (owner checks, filters,
// sorting, foreign keys) and backs unit tests and the agent evaluation suite.
package memstore

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

// Store keeps users and expenses in memory. It is safe for concurrent use.
type Store struct {
	mu       sync.Mutex
	clock    clock.Clock
	users    map[int64]user.User
	expenses map[int64]expense.Expense
	nextID   int64
}

// New creates an empty store; c stamps created_at and updated_at.
func New(c clock.Clock) *Store {
	return &Store{
		clock:    c,
		users:    map[int64]user.User{},
		expenses: map[int64]expense.Expense{},
		nextID:   1,
	}
}

// Users is the user.Repository view of a Store.
type Users struct{ s *Store }

// Expenses is the expense.Repository view of a Store.
type Expenses struct{ s *Store }

// Users returns the user repository backed by s.
func (s *Store) Users() Users { return Users{s} }

// Expenses returns the expense repository backed by s.
func (s *Store) Expenses() Expenses { return Expenses{s} }

var (
	_ expense.Repository = Expenses{}
	_ user.Repository    = Users{}
)

// Upsert implements user.Repository.
func (r Users) Upsert(_ context.Context, u user.User) (user.User, error) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[u.TelegramID] = u
	return u, nil
}

// List implements user.Repository.
func (r Users) List(_ context.Context) ([]user.User, error) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]user.User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	slices.SortFunc(out, func(a, b user.User) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.TelegramID, b.TelegramID))
	})
	return out, nil
}

// unknownUser mirrors the foreign key violation mapping of the PostgreSQL store.
func unknownUser() error {
	return &expense.ValidationError{Fields: map[string]string{"user_id": "Неизвестный член семьи"}}
}

// Create implements expense.Repository.
func (r Expenses) Create(_ context.Context, e expense.NewExpense) (expense.Expense, error) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[e.UserID]; !ok {
		return expense.Expense{}, unknownUser()
	}
	now := s.clock.Now().UTC()
	row := expense.Expense{
		ID:        s.nextID,
		UserID:    e.UserID,
		Amount:    e.Amount,
		Category:  e.Category,
		SpentAt:   e.SpentAt,
		Comment:   clonePtr(e.Comment),
		Source:    e.Source,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.nextID++
	s.expenses[row.ID] = row
	return s.withName(row), nil
}

// Get implements expense.Repository.
func (r Expenses) Get(_ context.Context, id int64, owner *int64) (expense.Expense, error) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.find(id, owner)
	if err != nil {
		return expense.Expense{}, err
	}
	return s.withName(row), nil
}

// Update implements expense.Repository.
func (r Expenses) Update(_ context.Context, id int64, owner *int64, in expense.UpdateInput) (expense.Expense, error) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.find(id, owner)
	if err != nil {
		return expense.Expense{}, err
	}
	if in.UserID != nil {
		if _, ok := s.users[*in.UserID]; !ok {
			return expense.Expense{}, unknownUser()
		}
		row.UserID = *in.UserID
	}
	if in.Amount != nil {
		row.Amount = *in.Amount
	}
	if in.Category != nil {
		row.Category = *in.Category
	}
	if in.SpentAt != nil {
		row.SpentAt = *in.SpentAt
	}
	if in.Comment != nil {
		row.Comment = nil
		if *in.Comment != "" {
			row.Comment = clonePtr(in.Comment)
		}
	}
	row.UpdatedAt = s.clock.Now().UTC()
	s.expenses[id] = row
	return s.withName(row), nil
}

// Delete implements expense.Repository.
func (r Expenses) Delete(_ context.Context, id int64, owner *int64) (expense.Expense, error) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.find(id, owner)
	if err != nil {
		return expense.Expense{}, err
	}
	delete(s.expenses, id)
	return s.withName(row), nil
}

// List implements expense.Repository. The filter is expected to be normalized.
func (r Expenses) List(_ context.Context, f expense.Filter) ([]expense.Expense, error) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := s.match(f)
	slices.SortFunc(rows, func(a, b expense.Expense) int {
		c := cmp.Or(compareBy(a, b, f.Sort), cmp.Compare(a.ID, b.ID))
		if f.Order == expense.OrderDesc {
			return -c
		}
		return c
	})
	if f.Offset >= len(rows) {
		return []expense.Expense{}, nil
	}
	rows = rows[f.Offset:]
	if f.Limit > 0 && len(rows) > f.Limit {
		rows = rows[:f.Limit]
	}
	for i := range rows {
		rows[i] = s.withName(rows[i])
	}
	return rows, nil
}

// Count implements expense.Repository.
func (r Expenses) Count(_ context.Context, f expense.Filter) (int, error) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.match(f)), nil
}

// Summary implements expense.Repository. Categories with equal totals
// keep their display order.
func (r Expenses) Summary(_ context.Context, f expense.Filter) (expense.Summary, error) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	var sum expense.Summary
	byCat := map[expense.Category]*expense.CategoryTotal{}
	for _, row := range s.match(f) {
		sum.Total += row.Amount
		sum.Count++
		ct, ok := byCat[row.Category]
		if !ok {
			ct = &expense.CategoryTotal{Category: row.Category}
			byCat[row.Category] = ct
		}
		ct.Total += row.Amount
		ct.Count++
	}
	sum.ByCategory = []expense.CategoryTotal{}
	for _, info := range expense.Categories() {
		if ct, ok := byCat[info.Code]; ok {
			sum.ByCategory = append(sum.ByCategory, *ct)
		}
	}
	slices.SortStableFunc(sum.ByCategory, func(a, b expense.CategoryTotal) int {
		return cmp.Compare(b.Total, a.Total)
	})
	return sum, nil
}

func (s *Store) find(id int64, owner *int64) (expense.Expense, error) {
	row, ok := s.expenses[id]
	if !ok || (owner != nil && row.UserID != *owner) {
		return expense.Expense{}, expense.ErrNotFound
	}
	return row, nil
}

func (s *Store) match(f expense.Filter) []expense.Expense {
	query := strings.ToLower(f.Query)
	var out []expense.Expense
	for _, row := range s.expenses {
		switch {
		case f.UserID != nil && row.UserID != *f.UserID,
			f.From != nil && row.SpentAt.Before(*f.From),
			f.To != nil && row.SpentAt.After(*f.To),
			f.Category != nil && row.Category != *f.Category,
			query != "" && (row.Comment == nil || !strings.Contains(strings.ToLower(*row.Comment), query)):
			continue
		}
		out = append(out, row)
	}
	return out
}

// withName prepares a row for returning: joins the user name and copies
// pointers, so callers cannot modify the stored data.
func (s *Store) withName(row expense.Expense) expense.Expense {
	row.Comment = clonePtr(row.Comment)
	row.UserName = s.users[row.UserID].Name
	return row
}

func compareBy(a, b expense.Expense, field expense.SortField) int {
	switch field {
	case expense.SortAmount:
		return cmp.Compare(a.Amount, b.Amount)
	case expense.SortCreatedAt:
		return a.CreatedAt.Compare(b.CreatedAt)
	default:
		return a.SpentAt.Compare(b.SpentAt)
	}
}

func clonePtr[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
