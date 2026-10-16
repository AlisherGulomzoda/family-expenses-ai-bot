package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/postgres/sqlcdb"
)

// ChatHistoryRepo implements agent.History on the chat_messages table.
// Each message is stored as its JSON form; the role is duplicated in a
// column for ad hoc queries.
type ChatHistoryRepo struct {
	pool *pgxpool.Pool
	q    *sqlcdb.Queries
}

// NewChatHistoryRepo creates a ChatHistoryRepo.
func NewChatHistoryRepo(pool *pgxpool.Pool) *ChatHistoryRepo {
	return &ChatHistoryRepo{pool: pool, q: sqlcdb.New(pool)}
}

// Append stores the messages of one turn in a single transaction, so a
// turn is either fully stored or not at all.
func (r *ChatHistoryRepo) Append(ctx context.Context, userID int64, msgs []llm.Message) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		for _, m := range msgs {
			content, err := json.Marshal(m)
			if err != nil {
				return fmt.Errorf("encode chat message: %w", err)
			}
			if err := q.AppendChatMessage(ctx, sqlcdb.AppendChatMessageParams{
				UserID: userID, Role: string(m.Role), Content: content,
			}); err != nil {
				return fmt.Errorf("append chat message: %w", err)
			}
		}
		return nil
	})
}

// Last returns up to n most recent messages in chronological order.
func (r *ChatHistoryRepo) Last(ctx context.Context, userID int64, n int) ([]llm.Message, error) {
	if n <= 0 {
		return []llm.Message{}, nil
	}
	rows, err := r.q.LastChatMessages(ctx, sqlcdb.LastChatMessagesParams{
		UserID: userID, RowLimit: int32(min(n, 1000)), //nolint:gosec // clamped
	})
	if err != nil {
		return nil, fmt.Errorf("load chat history: %w", err)
	}
	out := make([]llm.Message, len(rows))
	for i, raw := range rows {
		if err := json.Unmarshal(raw, &out[i]); err != nil {
			return nil, fmt.Errorf("decode chat message: %w", err)
		}
	}
	return out, nil
}
