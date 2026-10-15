package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/postgres/sqlcdb"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

// UserRepo implements user.Repository on PostgreSQL.
type UserRepo struct {
	q *sqlcdb.Queries
}

var _ user.Repository = (*UserRepo)(nil)

// NewUserRepo creates a UserRepo.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{q: sqlcdb.New(pool)}
}

// Upsert implements user.Repository.
func (r *UserRepo) Upsert(ctx context.Context, u user.User) (user.User, error) {
	row, err := r.q.UpsertUser(ctx, sqlcdb.UpsertUserParams{
		TelegramID: u.TelegramID,
		Name:       u.Name,
		Username:   u.Username,
	})
	if err != nil {
		return user.User{}, fmt.Errorf("upsert user %d: %w", u.TelegramID, err)
	}
	return user.User{TelegramID: row.TelegramID, Name: row.Name, Username: row.Username}, nil
}

// List implements user.Repository.
func (r *UserRepo) List(ctx context.Context) ([]user.User, error) {
	rows, err := r.q.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	out := make([]user.User, len(rows))
	for i, row := range rows {
		out[i] = user.User(row)
	}
	return out, nil
}
