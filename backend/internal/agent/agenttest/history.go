// Package agenttest holds the behaviour every agent.History must share.
// It runs against the in-memory store and against PostgreSQL.
package agenttest

import (
	"context"
	"reflect"
	"testing"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
)

// Users that newHistory must have registered (PostgreSQL has a foreign key).
const (
	UserA int64 = 100000001
	UserB int64 = 100000002
)

// RunHistoryContract runs the shared history tests. newHistory must return
// an empty history where UserA and UserB exist.
func RunHistoryContract(t *testing.T, newHistory func(t *testing.T) agent.History) {
	t.Run("round trip of a tool-calling turn", func(t *testing.T) {
		h := newHistory(t)
		ctx := context.Background()
		turn := []llm.Message{
			{Role: llm.RoleUser, Content: "кофе 350, такси 1200"},
			{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{
				{ID: "call_1", Name: "add_expense", Arguments: `{"amount":"350","category":"food"}`},
				{ID: "call_2", Name: "add_expense", Arguments: `{"amount": broken json`},
			}},
			{Role: llm.RoleTool, ToolCallID: "call_1", Content: `{"created":{"id":1}}`},
			{Role: llm.RoleTool, ToolCallID: "call_2", Content: `{"error":"bad json"}`},
			{Role: llm.RoleAssistant, Content: "Записал кофе — 350 ₽."},
		}
		if err := h.Append(ctx, UserA, turn); err != nil {
			t.Fatal(err)
		}
		got, err := h.Last(ctx, UserA, 100)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, turn) {
			t.Errorf("Last = %+v\nwant   %+v", got, turn)
		}
	})

	t.Run("last n in chronological order, per user", func(t *testing.T) {
		h := newHistory(t)
		ctx := context.Background()
		for _, text := range []string{"1", "2", "3", "4"} {
			if err := h.Append(ctx, UserA, []llm.Message{{Role: llm.RoleUser, Content: text}}); err != nil {
				t.Fatal(err)
			}
		}
		if err := h.Append(ctx, UserB, []llm.Message{{Role: llm.RoleUser, Content: "чужое"}}); err != nil {
			t.Fatal(err)
		}

		got, err := h.Last(ctx, UserA, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0].Content != "3" || got[1].Content != "4" {
			t.Errorf("Last(2) = %+v", got)
		}
		got, _ = h.Last(ctx, UserB, 10)
		if len(got) != 1 || got[0].Content != "чужое" {
			t.Errorf("UserB = %+v", got)
		}
		if got, _ := h.Last(ctx, UserA, 0); len(got) != 0 {
			t.Errorf("Last(0) = %+v", got)
		}
		if got, err := h.Last(ctx, 42, 10); err != nil || len(got) != 0 {
			t.Errorf("unknown user = %+v, %v", got, err)
		}
	})
}
