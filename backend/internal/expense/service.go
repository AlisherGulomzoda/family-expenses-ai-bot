package expense

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
)

// Repository stores expenses.
//
// owner restricts an operation to the expenses of one user: nil means
// "any user" (web panel), non-nil makes a foreign expense look missing
// (ErrNotFound). Update and Delete return the expense after/before the change.
type Repository interface {
	Create(ctx context.Context, e NewExpense) (Expense, error)
	Get(ctx context.Context, id int64, owner *int64) (Expense, error)
	Update(ctx context.Context, id int64, owner *int64, in UpdateInput) (Expense, error)
	Delete(ctx context.Context, id int64, owner *int64) (Expense, error)
	List(ctx context.Context, f Filter) ([]Expense, error)
	Count(ctx context.Context, f Filter) (int, error)
	Summary(ctx context.Context, f Filter) (Summary, error)
}

// Service is the expense use cases of the family head (web panel):
// every family member's expenses are visible and editable.
//
// The bot never uses Service directly: it gets a UserScope from ForUser.
type Service struct {
	repo  Repository
	clock clock.Clock
	loc   *time.Location
}

// NewService creates a Service. loc defines what "today" is.
func NewService(repo Repository, c clock.Clock, loc *time.Location) *Service {
	return &Service{repo: repo, clock: c, loc: loc}
}

func (s *Service) rules() rules {
	return rules{today: clock.Today(s.clock, s.loc)}
}

// Today returns the current date in the service time zone (midnight UTC).
func (s *Service) Today() time.Time {
	return clock.Today(s.clock, s.loc)
}

// List returns a page of expenses matching f and the total count.
func (s *Service) List(ctx context.Context, f Filter) (Page, error) {
	if err := f.normalize(); err != nil {
		return Page{}, err
	}
	items, err := s.repo.List(ctx, f)
	if err != nil {
		return Page{}, fmt.Errorf("list expenses: %w", err)
	}
	total, err := s.repo.Count(ctx, f)
	if err != nil {
		return Page{}, fmt.Errorf("count expenses: %w", err)
	}
	return Page{Items: items, Total: total}, nil
}

// Summary aggregates all expenses matching f.
func (s *Service) Summary(ctx context.Context, f Filter) (Summary, error) {
	if err := f.normalize(); err != nil {
		return Summary{}, err
	}
	sum, err := s.repo.Summary(ctx, f)
	if err != nil {
		return Summary{}, fmt.Errorf("summarize expenses: %w", err)
	}
	return sum, nil
}

// Get returns an expense by ID.
func (s *Service) Get(ctx context.Context, id int64) (Expense, error) {
	return s.repo.Get(ctx, id, nil)
}

// Create adds an expense for any family member. Its source is "web".
func (s *Service) Create(ctx context.Context, in CreateInput) (Expense, error) {
	errs := fieldErrors{}
	if in.UserID <= 0 {
		errs.add("user_id", "Выберите члена семьи")
	}
	if err := s.rules().validateDraft(&in.Draft); err != nil {
		mergeFields(errs, err)
	}
	if err := errs.err(); err != nil {
		return Expense{}, err
	}
	return s.repo.Create(ctx, newExpense(in.UserID, in.Draft, SourceWeb))
}

// Update changes an expense of any family member. The source never changes.
func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (Expense, error) {
	errs := fieldErrors{}
	if in.UserID != nil && *in.UserID <= 0 {
		errs.add("user_id", "Выберите члена семьи")
	}
	if err := s.rules().validatePatch(&in.Patch); err != nil {
		mergeFields(errs, err)
	}
	if err := errs.err(); err != nil {
		return Expense{}, err
	}
	if in.UserID == nil && in.Patch.IsEmpty() {
		return s.repo.Get(ctx, id, nil)
	}
	return s.repo.Update(ctx, id, nil, in)
}

// Delete removes an expense of any family member.
func (s *Service) Delete(ctx context.Context, id int64) error {
	_, err := s.repo.Delete(ctx, id, nil)
	return err
}

// ForUser returns the operations available to one family member.
func (s *Service) ForUser(userID int64) UserScope {
	return UserScope{svc: s, userID: userID}
}

// UserScope is the expense use cases of one family member (Telegram bot).
//
// The owner is fixed at construction and is not a parameter of any method:
// inputs (Draft, Patch) have no user field, filters are overridden, and every
// repository call carries the owner. A foreign expense is reported as
// ErrNotFound, so its existence is not revealed either.
type UserScope struct {
	svc    *Service
	userID int64
}

// UserID returns the owner of the scope.
func (u UserScope) UserID() int64 { return u.userID }

// Today returns the current date in the service time zone (midnight UTC).
func (u UserScope) Today() time.Time { return u.svc.Today() }

// List returns the user's expenses matching f. f.UserID is ignored.
func (u UserScope) List(ctx context.Context, f Filter) (Page, error) {
	f.UserID = &u.userID
	return u.svc.List(ctx, f)
}

// Summary aggregates the user's expenses matching f. f.UserID is ignored.
func (u UserScope) Summary(ctx context.Context, f Filter) (Summary, error) {
	f.UserID = &u.userID
	return u.svc.Summary(ctx, f)
}

// Get returns the user's expense by ID.
func (u UserScope) Get(ctx context.Context, id int64) (Expense, error) {
	return u.svc.repo.Get(ctx, id, &u.userID)
}

// Create adds an expense of the user. Its source is "bot".
func (u UserScope) Create(ctx context.Context, d Draft) (Expense, error) {
	if err := u.svc.rules().validateDraft(&d); err != nil {
		return Expense{}, err
	}
	return u.svc.repo.Create(ctx, newExpense(u.userID, d, SourceBot))
}

// Update changes the user's expense.
func (u UserScope) Update(ctx context.Context, id int64, p Patch) (Expense, error) {
	if err := u.svc.rules().validatePatch(&p); err != nil {
		return Expense{}, err
	}
	if p.IsEmpty() {
		return u.Get(ctx, id)
	}
	return u.svc.repo.Update(ctx, id, &u.userID, UpdateInput{Patch: p})
}

// Delete removes the user's expense and returns what was deleted,
// so the bot can tell the user exactly which expense is gone.
func (u UserScope) Delete(ctx context.Context, id int64) (Expense, error) {
	return u.svc.repo.Delete(ctx, id, &u.userID)
}

func newExpense(userID int64, d Draft, src Source) NewExpense {
	return NewExpense{
		UserID:   userID,
		Amount:   d.Amount,
		Category: d.Category,
		SpentAt:  d.SpentAt,
		Comment:  d.Comment,
		Source:   src,
	}
}

func mergeFields(dst fieldErrors, err error) {
	if ve := (*ValidationError)(nil); errors.As(err, &ve) {
		for k, v := range ve.Fields {
			dst.add(k, v)
		}
	}
}
