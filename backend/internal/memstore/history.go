package memstore

import (
	"context"
	"slices"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
)

// History is the agent.History view of a Store.
type History struct{ s *Store }

// History returns the conversation history backed by s.
func (s *Store) History() History { return History{s} }

// Append stores the messages of one turn.
func (r History) Append(_ context.Context, userID int64, msgs []llm.Message) error {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.history == nil {
		s.history = map[int64][]llm.Message{}
	}
	for _, m := range msgs {
		m.ToolCalls = slices.Clone(m.ToolCalls)
		s.history[userID] = append(s.history[userID], m)
	}
	return nil
}

// Last returns up to n most recent messages in chronological order.
func (r History) Last(_ context.Context, userID int64, n int) ([]llm.Message, error) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	all := s.history[userID]
	if n <= 0 {
		return []llm.Message{}, nil
	}
	start := max(len(all)-n, 0)
	out := make([]llm.Message, 0, len(all)-start)
	for _, m := range all[start:] {
		m.ToolCalls = slices.Clone(m.ToolCalls)
		out = append(out, m)
	}
	return out, nil
}
