//go:build live

package llm_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
)

// TestLiveToolCall calls the real provider from the environment
// (LLM_BASE_URL, LLM_API_KEY, LLM_MODEL):
//
//	go test -tags live -run Live ./internal/llm/
func TestLiveToolCall(t *testing.T) {
	key := os.Getenv("LLM_API_KEY")
	if key == "" {
		t.Skip("LLM_API_KEY is not set")
	}
	c := llm.NewOpenAI(llm.OpenAIConfig{
		BaseURL:    envOr("LLM_BASE_URL", "https://api.deepseek.com"),
		APIKey:     key,
		Model:      envOr("LLM_MODEL", "deepseek-chat"),
		Timeout:    60 * time.Second,
		MaxRetries: 1,
	})

	tools := []llm.ToolDef{{
		Name:        "add_expense",
		Description: "Записать трату пользователя.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"amount":   {"type": "string", "description": "Сумма в рублях, например \"350\" или \"349.90\""},
				"category": {"type": "string", "enum": ["food","transport","housing","entertainment","health","shopping","other"]},
				"comment":  {"type": "string"}
			},
			"required": ["amount", "category"]
		}`),
	}}
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "Ты помощник учёта расходов. Записывай траты через инструменты."},
		{Role: llm.RoleUser, Content: "кофе 350"},
	}

	resp, err := c.Chat(t.Context(), llm.Request{Messages: msgs, Tools: tools})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("finish=%s tokens=%d/%d calls=%+v", resp.FinishReason, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Message.ToolCalls)
	if len(resp.Message.ToolCalls) != 1 || resp.Message.ToolCalls[0].Name != "add_expense" {
		t.Fatalf("want one add_expense call, got %+v", resp.Message)
	}
	var args struct{ Amount, Category string }
	if err := json.Unmarshal([]byte(resp.Message.ToolCalls[0].Arguments), &args); err != nil {
		t.Fatalf("arguments are not JSON: %v", err)
	}
	if !strings.HasPrefix(args.Amount, "350") || args.Category != "food" {
		t.Errorf("arguments = %+v", args)
	}

	// Second turn: send the tool result back and expect a text answer.
	msgs = append(msgs, resp.Message, llm.Message{
		Role: llm.RoleTool, ToolCallID: resp.Message.ToolCalls[0].ID,
		Content: `{"id":1,"amount":"350 ₽","category":"Еда","spent_at":"2026-10-16","comment":"кофе"}`,
	})
	resp, err = c.Chat(t.Context(), llm.Request{Messages: msgs, Tools: tools})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("final answer: %q", resp.Message.Content)
	if resp.Message.Content == "" || len(resp.Message.ToolCalls) != 0 {
		t.Errorf("want a text answer, got %+v", resp.Message)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
