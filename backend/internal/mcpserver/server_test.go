package mcpserver_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent/tools"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/mcpserver"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/memstore"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

const (
	alisher int64 = 100000001
	madina  int64 = 100000002
)

// 2026-10-16 10:00 in Moscow.
var now = clock.Fixed(time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC))

type env struct {
	svc     *expense.Service
	session *mcp.ClientSession
}

// newEnv connects an MCP client to a server acting as Alisher, over in-memory
// transports, with one expense of Madina in the store.
func newEnv(t *testing.T) (*env, expense.Expense) {
	t.Helper()
	ctx := context.Background()
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	store := memstore.New(now)
	for _, u := range []user.User{{TelegramID: alisher, Name: "Алишер"}, {TelegramID: madina, Name: "Мадина"}} {
		if _, err := store.Users().Upsert(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	svc := expense.NewService(store.Expenses(), now, moscow)
	taxi := "такси"
	foreign, err := svc.ForUser(madina).Create(ctx, expense.Draft{Amount: 99900, Category: expense.CategoryTransport, Comment: &taxi})
	if err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := mcpserver.New(tools.Default(), svc.ForUser(alisher), "Алишер", log)
	serverT, clientT := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return &env{svc: svc, session: cs}, foreign
}

func (e *env) call(t *testing.T, name string, args any) (string, bool) {
	t.Helper()
	res, err := e.session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if len(res.Content) != 1 {
		t.Fatalf("call %s: %d content items, want 1", name, len(res.Content))
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("call %s: content is %T, want text", name, res.Content[0])
	}
	return text.Text, res.IsError
}

func TestListsRegistryToolsWithTheirSchemas(t *testing.T) {
	e, _ := newEnv(t)
	res, err := e.session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := tools.Default().Tools()
	if len(res.Tools) != len(want) {
		t.Fatalf("got %d tools, want %d", len(res.Tools), len(want))
	}
	// The SDK lists tools by name, not in registration order.
	byName := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		byName[tool.Name] = tool
	}
	for _, w := range want {
		got, ok := byName[w.Name]
		if !ok {
			t.Errorf("tool %s is not listed", w.Name)
			continue
		}
		if got.Description != w.Description {
			t.Errorf("%s: description differs from the registry", w.Name)
		}
		if g, x := normalize(t, got.InputSchema), normalize(t, w.Schema); g != x {
			t.Errorf("%s schema differs from the registry:\n got %s\nwant %s", w.Name, g, x)
		}
		if got.Annotations == nil || got.Annotations.ReadOnlyHint != w.ReadOnly {
			t.Errorf("%s: read-only hint should be %v", w.Name, w.ReadOnly)
		}
	}
}

func TestInstructionsNameTheUserAndToday(t *testing.T) {
	e, _ := newEnv(t)
	got := e.session.InitializeResult().Instructions
	for _, want := range []string{"Алишер", "Europe/Moscow", "2026-10-16"} {
		if !strings.Contains(got, want) {
			t.Errorf("instructions do not mention %q:\n%s", want, got)
		}
	}
}

func TestCallsActOnTheConfiguredUser(t *testing.T) {
	e, _ := newEnv(t)
	ctx := context.Background()

	out, isErr := e.call(t, "add_expense", map[string]any{"amount": "350", "category": "food", "comment": "кофе"})
	if isErr {
		t.Fatalf("add_expense failed: %s", out)
	}
	page, err := e.svc.ForUser(alisher).List(ctx, expense.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Comment == nil || *page.Items[0].Comment != "кофе" ||
		!page.Items[0].SpentAt.Equal(e.svc.Today()) {
		t.Fatalf("Alisher's expenses = %+v, want today's кофе", page.Items)
	}

	out, isErr = e.call(t, "summarize_expenses", map[string]any{})
	if isErr || !strings.Contains(out, `"total":"350`) || strings.Contains(out, "999") {
		t.Errorf("summary = %s (error %v), want only Alisher's 350", out, isErr)
	}
}

func TestErrorsAreToolResults(t *testing.T) {
	e, foreign := newEnv(t)

	// Bad arguments: the model gets the reason and can fix the call.
	out, isErr := e.call(t, "add_expense", map[string]any{"amount": "-5", "category": "food"})
	if !isErr || !strings.Contains(out, `"error"`) {
		t.Errorf("negative amount: got %s (error %v), want an error result", out, isErr)
	}

	// Madina's expense does not exist for Alisher.
	out, isErr = e.call(t, "delete_expense", map[string]any{"id": foreign.ID})
	if !isErr || !strings.Contains(out, "не найдена") {
		t.Errorf("foreign delete: got %s (error %v), want not found", out, isErr)
	}
	if _, err := e.svc.ForUser(madina).Get(context.Background(), foreign.ID); err != nil {
		t.Errorf("Madina's expense must stay: %v", err)
	}
}

// normalize re-encodes a JSON value so that formatting does not matter.
func normalize(t *testing.T, v any) string {
	t.Helper()
	raw, ok := v.(json.RawMessage)
	if !ok {
		var err error
		if raw, err = json.Marshal(v); err != nil {
			t.Fatal(err)
		}
	}
	var x any
	if err := json.Unmarshal(raw, &x); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(x)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
