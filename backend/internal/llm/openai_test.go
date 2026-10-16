package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var addExpense = ToolDef{
	Name:        "add_expense",
	Description: "Add an expense",
	Parameters:  json.RawMessage(`{"type":"object","properties":{"amount":{"type":"string"}},"required":["amount"]}`),
}

// fakeAPI serves scripted HTTP responses and records request bodies.
type fakeAPI struct {
	t        *testing.T
	calls    atomic.Int32
	bodies   []map[string]any
	headers  []http.Header
	paths    []string
	handlers []http.HandlerFunc
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n := int(f.calls.Add(1)) - 1
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		f.t.Errorf("request body is not JSON: %s", raw)
	}
	f.bodies = append(f.bodies, body)
	f.headers = append(f.headers, r.Header.Clone())
	f.paths = append(f.paths, r.URL.Path)
	if n >= len(f.handlers) {
		f.t.Errorf("unexpected request #%d", n+1)
		http.Error(w, "no more responses", http.StatusTeapot)
		return
	}
	f.handlers[n](w, r)
}

func respond(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

const textReply = `{"choices":[{"message":{"role":"assistant","content":"Записал: кофе 350 ₽"},"finish_reason":"stop"}],
	"usage":{"prompt_tokens":120,"completion_tokens":12}}`

const toolReply = `{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":[
	{"id":"call_1","type":"function","function":{"name":"add_expense","arguments":"{\"amount\":\"350\"}"}},
	{"id":"call_2","type":"function","function":{"name":"add_expense","arguments":"{\"amount\":\"1200\"}"}}
]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":200,"completion_tokens":40}}`

func newTestClient(t *testing.T, cfg OpenAIConfig, handlers ...http.HandlerFunc) (*OpenAI, *fakeAPI) {
	t.Helper()
	api := &fakeAPI{t: t, handlers: handlers}
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)
	if cfg.BaseURL == "" {
		cfg.BaseURL = srv.URL
	} else {
		cfg.BaseURL = srv.URL + cfg.BaseURL
	}
	cfg.APIKey = "sk-test"
	cfg.Model = "deepseek-chat"
	c := NewOpenAI(cfg)
	c.backoff = func(int) time.Duration { return time.Millisecond }
	return c, api
}

func TestChatSendsRequestAndParsesText(t *testing.T) {
	c, api := newTestClient(t, OpenAIConfig{BaseURL: "/v1/", Temperature: 0.1}, respond(200, textReply))

	resp, err := c.Chat(t.Context(), Request{
		Messages: []Message{{Role: RoleSystem, Content: "system prompt"}, {Role: RoleUser, Content: "кофе 350"}},
		Tools:    []ToolDef{addExpense},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message.Role != RoleAssistant || resp.Message.Content != "Записал: кофе 350 ₽" || resp.FinishReason != "stop" {
		t.Errorf("response = %+v", resp)
	}
	if resp.Usage != (Usage{PromptTokens: 120, CompletionTokens: 12}) {
		t.Errorf("usage = %+v", resp.Usage)
	}

	if api.paths[0] != "/v1/chat/completions" {
		t.Errorf("path = %q (trailing slash of the base URL must be handled)", api.paths[0])
	}
	if got := api.headers[0].Get("Authorization"); got != "Bearer sk-test" {
		t.Errorf("Authorization = %q", got)
	}
	body := api.bodies[0]
	if body["model"] != "deepseek-chat" || body["temperature"] != 0.1 || body["stream"] != false || body["tool_choice"] != "auto" {
		t.Errorf("request = %v", body)
	}
	tool := body["tools"].([]any)[0].(map[string]any)
	fn := tool["function"].(map[string]any)
	if tool["type"] != "function" || fn["name"] != "add_expense" || fn["parameters"].(map[string]any)["type"] != "object" {
		t.Errorf("tool = %v", tool)
	}
	msgs := body["messages"].([]any)
	if len(msgs) != 2 || msgs[1].(map[string]any)["content"] != "кофе 350" {
		t.Errorf("messages = %v", msgs)
	}
}

func TestChatToolCallsRoundTrip(t *testing.T) {
	c, api := newTestClient(t, OpenAIConfig{}, respond(200, toolReply), respond(200, textReply))

	resp, err := c.Chat(t.Context(), Request{Messages: []Message{{Role: RoleUser, Content: "кофе 350, такси 1200"}}})
	if err != nil {
		t.Fatal(err)
	}
	calls := resp.Message.ToolCalls
	if len(calls) != 2 || calls[0].ID != "call_1" || calls[0].Name != "add_expense" || calls[1].Arguments != `{"amount":"1200"}` {
		t.Fatalf("tool calls = %+v", calls)
	}
	if resp.Message.Content != "" || resp.FinishReason != "tool_calls" {
		t.Errorf("null content must become empty: %+v", resp)
	}
	if _, ok := api.bodies[0]["tools"]; ok {
		t.Error("no tools in the request means no tools field")
	}

	// The assistant's tool calls and the tool results go back to the API.
	_, err = c.Chat(t.Context(), Request{Messages: []Message{
		{Role: RoleUser, Content: "кофе 350"},
		resp.Message,
		{Role: RoleTool, ToolCallID: "call_1", Content: `{"id":1}`},
	}})
	if err != nil {
		t.Fatal(err)
	}
	msgs := api.bodies[1]["messages"].([]any)
	assistant := msgs[1].(map[string]any)
	tc := assistant["tool_calls"].([]any)[0].(map[string]any)
	if tc["id"] != "call_1" || tc["type"] != "function" || tc["function"].(map[string]any)["arguments"] != `{"amount":"350"}` {
		t.Errorf("assistant message = %v", assistant)
	}
	if assistant["content"] != "" {
		t.Errorf("assistant content = %v, want empty string", assistant["content"])
	}
	toolMsg := msgs[2].(map[string]any)
	if toolMsg["role"] != "tool" || toolMsg["tool_call_id"] != "call_1" || toolMsg["content"] != `{"id":1}` {
		t.Errorf("tool message = %v", toolMsg)
	}
}

func TestChatRetries(t *testing.T) {
	t.Run("rate limit then success", func(t *testing.T) {
		c, api := newTestClient(t, OpenAIConfig{MaxRetries: 2},
			respond(429, `{"error":{"message":"Rate limit reached"}}`),
			respond(200, textReply))
		if _, err := c.Chat(t.Context(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}}); err != nil {
			t.Fatal(err)
		}
		if api.calls.Load() != 2 {
			t.Errorf("calls = %d, want 2", api.calls.Load())
		}
	})

	t.Run("server errors exhaust retries", func(t *testing.T) {
		c, api := newTestClient(t, OpenAIConfig{MaxRetries: 2},
			respond(500, `oops`), respond(502, `bad gateway`), respond(503, `{"error":{"message":"Service is too busy"}}`))
		_, err := c.Chat(t.Context(), Request{})
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 503 || apiErr.Message != "Service is too busy" {
			t.Fatalf("err = %v", err)
		}
		if api.calls.Load() != 3 {
			t.Errorf("calls = %d, want 3", api.calls.Load())
		}
	})

	t.Run("client errors are not retried", func(t *testing.T) {
		for _, status := range []int{400, 401, 402, 422} {
			c, api := newTestClient(t, OpenAIConfig{MaxRetries: 2},
				respond(status, `{"error":{"message":"Insufficient Balance"}}`))
			_, err := c.Chat(t.Context(), Request{})
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != status || !strings.Contains(err.Error(), "Insufficient Balance") {
				t.Errorf("status %d: err = %v", status, err)
			}
			if api.calls.Load() != 1 {
				t.Errorf("status %d: calls = %d, want 1", status, api.calls.Load())
			}
		}
	})

	t.Run("attempt timeout is retried", func(t *testing.T) {
		slow := func(_ http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-time.After(time.Second):
			}
		}
		c, api := newTestClient(t, OpenAIConfig{MaxRetries: 1, Timeout: 50 * time.Millisecond}, slow, respond(200, textReply))
		if _, err := c.Chat(t.Context(), Request{}); err != nil {
			t.Fatalf("err = %v", err)
		}
		if api.calls.Load() != 2 {
			t.Errorf("calls = %d, want 2", api.calls.Load())
		}
	})

	t.Run("canceled caller is not retried", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancelling := func(w http.ResponseWriter, _ *http.Request) {
			cancel()
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		c, api := newTestClient(t, OpenAIConfig{MaxRetries: 3}, cancelling)
		if _, err := c.Chat(ctx, Request{}); err == nil {
			t.Fatal("expected an error")
		}
		if api.calls.Load() != 1 {
			t.Errorf("calls = %d, want 1", api.calls.Load())
		}
	})
}

func TestChatBadResponses(t *testing.T) {
	for name, body := range map[string]string{
		"no choices":   `{"choices":[]}`,
		"invalid json": `{"choices":`,
	} {
		c, _ := newTestClient(t, OpenAIConfig{}, respond(200, body))
		if _, err := c.Chat(t.Context(), Request{}); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestParseRetryAfterAndBackoff(t *testing.T) {
	if parseRetryAfter("3") != 3*time.Second || parseRetryAfter("") != 0 || parseRetryAfter("soon") != 0 || parseRetryAfter("600") != 30*time.Second {
		t.Error("parseRetryAfter")
	}
	for n := 1; n <= 10; n++ {
		if d := exponentialBackoff(n); d <= 0 || d > 10*time.Second {
			t.Errorf("backoff(%d) = %v", n, d)
		}
	}
}

func TestMessageJSONIsStable(t *testing.T) {
	// The history is stored as JSON: keep the field names.
	m := Message{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "add_expense", Arguments: `{"amount":"350"}`}}}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"role":"assistant","tool_calls":[{"id":"c1","name":"add_expense","arguments":"{\"amount\":\"350\"}"}]}`
	if string(raw) != want {
		t.Errorf("json = %s\nwant  %s", raw, want)
	}
	var back Message
	if err := json.Unmarshal(raw, &back); err != nil || back.ToolCalls[0].Arguments != m.ToolCalls[0].Arguments {
		t.Errorf("round trip = %+v, %v", back, err)
	}
}
