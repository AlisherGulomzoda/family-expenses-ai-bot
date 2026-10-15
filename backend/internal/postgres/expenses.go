package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/postgres/sqlcdb"
)

// ExpenseRepo implements expense.Repository on PostgreSQL.
//
// The owner argument of Get, Update and Delete goes straight into the WHERE
// clause (owner IS NULL OR user_id = owner): with an owner set, another
// user's expense is simply not found by the query.
type ExpenseRepo struct {
	q *sqlcdb.Queries
}

var _ expense.Repository = (*ExpenseRepo)(nil)

// NewExpenseRepo creates an ExpenseRepo.
func NewExpenseRepo(pool *pgxpool.Pool) *ExpenseRepo {
	return &ExpenseRepo{q: sqlcdb.New(pool)}
}

// Create implements expense.Repository.
func (r *ExpenseRepo) Create(ctx context.Context, e expense.NewExpense) (expense.Expense, error) {
	row, err := r.q.CreateExpense(ctx, sqlcdb.CreateExpenseParams{
		UserID:        e.UserID,
		AmountKopecks: int64(e.Amount),
		Category:      string(e.Category),
		SpentAt:       e.SpentAt,
		Comment:       e.Comment,
		Source:        string(e.Source),
	})
	if isForeignKeyViolation(err) {
		return expense.Expense{}, unknownUser()
	}
	if err != nil {
		return expense.Expense{}, fmt.Errorf("create expense: %w", err)
	}
	return toExpense(sqlcdb.GetExpenseRow(row)), nil
}

// Get implements expense.Repository.
func (r *ExpenseRepo) Get(ctx context.Context, id int64, owner *int64) (expense.Expense, error) {
	row, err := r.q.GetExpense(ctx, sqlcdb.GetExpenseParams{ID: id, Owner: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		return expense.Expense{}, expense.ErrNotFound
	}
	if err != nil {
		return expense.Expense{}, fmt.Errorf("get expense %d: %w", id, err)
	}
	return toExpense(row), nil
}

// Update implements expense.Repository.
func (r *ExpenseRepo) Update(ctx context.Context, id int64, owner *int64, in expense.UpdateInput) (expense.Expense, error) {
	params := sqlcdb.UpdateExpenseParams{
		ID:        id,
		Owner:     owner,
		NewUserID: in.UserID,
		SpentAt:   in.SpentAt,
		Comment:   in.Comment,
	}
	if in.Amount != nil {
		v := int64(*in.Amount)
		params.AmountKopecks = &v
	}
	if in.Category != nil {
		v := string(*in.Category)
		params.Category = &v
	}

	row, err := r.q.UpdateExpense(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		// The update joins users on the new owner, so an unknown new owner
		// also yields no rows. Tell the two cases apart.
		if in.UserID != nil {
			exists, existsErr := r.q.UserExists(ctx, *in.UserID)
			if existsErr != nil {
				return expense.Expense{}, fmt.Errorf("check user %d: %w", *in.UserID, existsErr)
			}
			if !exists {
				return expense.Expense{}, unknownUser()
			}
		}
		return expense.Expense{}, expense.ErrNotFound
	}
	if err != nil {
		return expense.Expense{}, fmt.Errorf("update expense %d: %w", id, err)
	}
	return toExpense(sqlcdb.GetExpenseRow(row)), nil
}

// Delete implements expense.Repository.
func (r *ExpenseRepo) Delete(ctx context.Context, id int64, owner *int64) (expense.Expense, error) {
	row, err := r.q.DeleteExpense(ctx, sqlcdb.DeleteExpenseParams{ID: id, Owner: owner})
	if errors.Is(err, pgx.ErrNoRows) {
		return expense.Expense{}, expense.ErrNotFound
	}
	if err != nil {
		return expense.Expense{}, fmt.Errorf("delete expense %d: %w", id, err)
	}
	return toExpense(sqlcdb.GetExpenseRow(row)), nil
}

// List implements expense.Repository. The filter must be normalized.
func (r *ExpenseRepo) List(ctx context.Context, f expense.Filter) ([]expense.Expense, error) {
	w := where(f)
	rows, err := r.q.ListExpenses(ctx, sqlcdb.ListExpensesParams{
		UserID:    w.UserID,
		FromDate:  w.FromDate,
		ToDate:    w.ToDate,
		Category:  w.Category,
		Query:     w.Query,
		SortField: string(f.Sort),
		SortOrder: string(f.Order),
		RowLimit:  int32(min(f.Limit, expense.MaxLimit)), //nolint:gosec // bounded by MaxLimit
		RowOffset: int32(min(f.Offset, 1<<31-1)),         //nolint:gosec // clamped to int32
	})
	if err != nil {
		return nil, fmt.Errorf("list expenses: %w", err)
	}
	out := make([]expense.Expense, len(rows))
	for i, row := range rows {
		out[i] = toExpense(sqlcdb.GetExpenseRow(row))
	}
	return out, nil
}

// Count implements expense.Repository.
func (r *ExpenseRepo) Count(ctx context.Context, f expense.Filter) (int, error) {
	n, err := r.q.CountExpenses(ctx, where(f))
	if err != nil {
		return 0, fmt.Errorf("count expenses: %w", err)
	}
	return int(n), nil
}

// Summary implements expense.Repository.
func (r *ExpenseRepo) Summary(ctx context.Context, f expense.Filter) (expense.Summary, error) {
	w := where(f)
	rows, err := r.q.SummarizeExpenses(ctx, sqlcdb.SummarizeExpensesParams(w))
	if err != nil {
		return expense.Summary{}, fmt.Errorf("summarize expenses: %w", err)
	}
	sum := expense.Summary{ByCategory: make([]expense.CategoryTotal, len(rows))}
	for i, row := range rows {
		sum.ByCategory[i] = expense.CategoryTotal{
			Category: expense.Category(row.Category),
			Total:    expense.Money(row.TotalKopecks),
			Count:    int(row.ExpenseCount),
		}
		sum.Total += expense.Money(row.TotalKopecks)
		sum.Count += int(row.ExpenseCount)
	}
	return sum, nil
}

// where converts the filter conditions shared by list, count and summary.
func where(f expense.Filter) sqlcdb.CountExpensesParams {
	p := sqlcdb.CountExpensesParams{
		UserID:   f.UserID,
		FromDate: f.From,
		ToDate:   f.To,
	}
	if f.Category != nil {
		c := string(*f.Category)
		p.Category = &c
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		q = escapeLike(q)
		p.Query = &q
	}
	return p
}

// escapeLike makes % and _ in user input match literally in ILIKE
// (backslash is the default escape character in PostgreSQL).
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func toExpense(row sqlcdb.GetExpenseRow) expense.Expense {
	return expense.Expense{
		ID:        row.ID,
		UserID:    row.UserID,
		UserName:  row.UserName,
		Amount:    expense.Money(row.AmountKopecks),
		Category:  expense.Category(row.Category),
		SpentAt:   row.SpentAt.UTC(),
		Comment:   row.Comment,
		Source:    expense.Source(row.Source),
		CreatedAt: row.CreatedAt.UTC(),
		UpdatedAt: row.UpdatedAt.UTC(),
	}
}

func unknownUser() error {
	return &expense.ValidationError{Fields: map[string]string{"user_id": "Неизвестный член семьи"}}
}
