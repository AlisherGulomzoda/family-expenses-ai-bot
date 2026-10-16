package agent

import (
	"context"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
)

// History stores the conversation of each user with the agent.
type History interface {
	// Append atomically stores the messages of one turn, in order.
	Append(ctx context.Context, userID int64, msgs []llm.Message) error
	// Last returns up to n most recent messages in chronological order.
	Last(ctx context.Context, userID int64, n int) ([]llm.Message, error)
}

// trimToTurn drops leading messages until the first user message, so the
// window never starts in the middle of a turn: an assistant tool call
// without its results (or results without the call) is rejected by
// chat APIs. Turns are stored atomically, so the tail is always complete.
func trimToTurn(msgs []llm.Message) []llm.Message {
	for i, m := range msgs {
		if m.Role == llm.RoleUser {
			return msgs[i:]
		}
	}
	return nil
}
