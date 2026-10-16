package agent_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent/agenttest"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent/tools"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm/llmtest"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/memstore"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

var (
	alisher = user.User{TelegramID: 100000001, Name: "Алишер"}
	// 2026-10-16 10:00 in Moscow.
	now = clock.Fixed(time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC))
)

type env struct {
	agent   *agent.Agent
	llm     *llmtest.Client
	store   *memstore.Store
	scope   expense.UserScope
	history memstore.History
}

func newEnv(t *testing.T, cfg agent.Config, steps ...llmtest.Step) *env {
	t.Helper()
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	store := memstore.New(now)
	if _, err := store.Users().Upsert(context.Background(), alisher); err != nil {
		t.Fatal(err)
	}
	svc := expense.NewService(store.Expenses(), now, moscow)
	fake := llmtest.New(steps...)
	if cfg.MaxSteps == 0 {
		cfg.MaxSteps = 5
	}
	if cfg.HistoryLimit == 0 {
		cfg.HistoryLimit = 20
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &env{
		agent:   agent.New(fake, tools.Default(), svc, store.History(), cfg, log),
		llm:     fake,
		store:   store,
		scope:   svc.ForUser(alisher.TelegramID),
		history: store.History(),
	}
}

func call(id, name, args string) llm.ToolCall {
	return llm.ToolCall{ID: id, Name: name, Arguments: args}
}

func (e *env) stored(t *testing.T) []llm.Message {
	t.Helper()
	msgs, err := e.history.Last(context.Background(), alisher.TelegramID, 100)
	if err != nil {
		t.Fatal(err)
	}
	return msgs
}

func roles(msgs []llm.Message) string {
	var b []string
	for _, m := range msgs {
		b = append(b, string(m.Role))
	}
	return strings.Join(b, ",")
}

func TestAddExpenseTurn(t *testing.T) {
	e := newEnv(t, agent.Config{},
		llmtest.Call(call("c1", "add_expense", `{"amount":"350","category":"food","comment":"кофе"}`)),
		llmtest.Reply("Записал: кофе — 350 ₽, еда."),
	)

	reply, err := e.agent.Handle(context.Background(), alisher, "кофе 350")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "Записал: кофе — 350 ₽, еда." {
		t.Errorf("reply = %q", reply)
	}

	page, _ := e.scope.List(context.Background(), expense.Filter{})
	if page.Total != 1 || page.Items[0].Amount != 35000 || page.Items[0].Source != expense.SourceBot {
		t.Errorf("stored expenses = %+v", page)
	}

	reqs := e.llm.Requests()
	if len(reqs) != 2 {
		t.Fatalf("llm calls = %d, want 2", len(reqs))
	}
	first := reqs[0]
	if first.Messages[0].Role != llm.RoleSystem || !strings.Contains(first.Messages[0].Content, "2026-10-16") ||
		!strings.Contains(first.Messages[0].Content, "Алишер") {
		t.Errorf("system prompt = %q", first.Messages[0].Content)
	}
	if last := first.Messages[len(first.Messages)-1]; last.Role != llm.RoleUser || last.Content != "кофе 350" {
		t.Errorf("last message = %+v", last)
	}
	if len(first.Tools) != 5 {
		t.Errorf("tools = %d, want 5", len(first.Tools))
	}
	// The tool result goes back with the id of its call.
	toolMsg := reqs[1].Messages[len(reqs[1].Messages)-1]
	if toolMsg.Role != llm.RoleTool || toolMsg.ToolCallID != "c1" || !strings.Contains(toolMsg.Content, `"created"`) {
		t.Errorf("tool message = %+v", toolMsg)
	}

	if got := roles(e.stored(t)); got != "user,assistant,tool,assistant" {
		t.Errorf("history = %s", got)
	}
}

func TestSeveralToolCallsInOneStep(t *testing.T) {
	e := newEnv(t, agent.Config{},
		llmtest.Call(
			call("c1", "add_expense", `{"amount":"350","category":"food","comment":"кофе"}`),
			call("c2", "add_expense", `{"amount":"500","category":"transport","comment":"такси"}`),
		),
		llmtest.Reply("Записал две траты."),
	)
	if _, err := e.agent.Handle(context.Background(), alisher, "кофе 350, такси 500"); err != nil {
		t.Fatal(err)
	}
	sum, _ := e.scope.Summary(context.Background(), expense.Filter{})
	if sum.Total != 85000 || sum.Count != 2 {
		t.Errorf("summary = %+v", sum)
	}
	msgs := e.llm.Requests()[1].Messages
	n := len(msgs)
	if msgs[n-2].ToolCallID != "c1" || msgs[n-1].ToolCallID != "c2" {
		t.Errorf("tool results order: %+v", msgs[n-2:])
	}
}

func TestToolErrorGoesBackToTheModel(t *testing.T) {
	e := newEnv(t, agent.Config{},
		llmtest.Call(call("c1", "add_expense", `{"category":"food","comment":"кофе"}`)),
		llmtest.Reply("Сколько стоил кофе?"),
	)
	reply, err := e.agent.Handle(context.Background(), alisher, "купил кофе")
	if err != nil || reply != "Сколько стоил кофе?" {
		t.Fatalf("reply = %q, %v", reply, err)
	}
	toolMsg := e.llm.Requests()[1].Messages[len(e.llm.Requests()[1].Messages)-1]
	if !strings.Contains(toolMsg.Content, `"error"`) {
		t.Errorf("tool error not passed to the model: %s", toolMsg.Content)
	}
	if page, _ := e.scope.List(context.Background(), expense.Filter{}); page.Total != 0 {
		t.Error("an expense without amount was stored")
	}
}

func TestHistoryIsSentNextTurn(t *testing.T) {
	e := newEnv(t, agent.Config{},
		llmtest.Call(call("c1", "add_expense", `{"amount":"350","category":"food","comment":"кофе"}`)),
		llmtest.Reply("Записал кофе."),
		llmtest.Reply("Исправил на 450 ₽."),
	)
	ctx := context.Background()
	if _, err := e.agent.Handle(ctx, alisher, "кофе 350"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.agent.Handle(ctx, alisher, "последнее было не 350, а 450"); err != nil {
		t.Fatal(err)
	}
	third := e.llm.Requests()[2].Messages
	// system + previous turn (4) + new user message
	if got := roles(third); got != "system,user,assistant,tool,assistant,user" {
		t.Errorf("request roles = %s", got)
	}
	if third[1].Content != "кофе 350" {
		t.Errorf("previous turn is missing: %+v", third[1])
	}
}

func TestHistoryWindowStartsAtATurn(t *testing.T) {
	// HistoryLimit 3 cuts the stored turn (user, assistant, tool, assistant)
	// after the user message: the partial turn must be dropped entirely.
	e := newEnv(t, agent.Config{HistoryLimit: 3},
		llmtest.Call(call("c1", "add_expense", `{"amount":"350","category":"food"}`)),
		llmtest.Reply("Записал."),
		llmtest.Reply("Привет!"),
	)
	ctx := context.Background()
	if _, err := e.agent.Handle(ctx, alisher, "кофе 350"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.agent.Handle(ctx, alisher, "привет"); err != nil {
		t.Fatal(err)
	}
	if got := roles(e.llm.Requests()[2].Messages); got != "system,user" {
		t.Errorf("request roles = %s, want the partial turn dropped", got)
	}
}

func TestMaxSteps(t *testing.T) {
	loop := llmtest.Call(call("c", "list_expenses", `{}`))
	e := newEnv(t, agent.Config{MaxSteps: 2}, loop, loop, loop)

	reply, err := e.agent.Handle(context.Background(), alisher, "покажи траты")
	if err == nil || reply != agent.ReplyTooManySteps {
		t.Fatalf("reply = %q, err = %v", reply, err)
	}
	if n := len(e.llm.Requests()); n != 2 {
		t.Errorf("llm calls = %d, want 2", n)
	}
	// The stored turn is closed with an assistant answer.
	if got := roles(e.stored(t)); got != "user,assistant,tool,assistant,tool,assistant" {
		t.Errorf("history = %s", got)
	}
}

func TestLLMFailure(t *testing.T) {
	e := newEnv(t, agent.Config{}, llmtest.Fail(errors.New("402 Insufficient Balance")))
	reply, err := e.agent.Handle(context.Background(), alisher, "кофе 350")
	if err == nil || reply != agent.ReplyLLMUnavailable {
		t.Fatalf("reply = %q, err = %v", reply, err)
	}
	if len(e.stored(t)) != 0 {
		t.Error("a failed turn must not be stored")
	}
}

func TestEmptyReply(t *testing.T) {
	e := newEnv(t, agent.Config{}, llmtest.Reply(""))
	reply, err := e.agent.Handle(context.Background(), alisher, "???")
	if err != nil || !strings.Contains(reply, "кофе 350") {
		t.Errorf("reply = %q, %v", reply, err)
	}
}

func TestMessagesOfOneUserDoNotInterleave(t *testing.T) {
	steps := make([]llmtest.Step, 0, 20)
	for range 10 {
		steps = append(steps, llmtest.Call(call("c", "list_expenses", `{}`)), llmtest.Reply("ok"))
	}
	e := newEnv(t, agent.Config{}, steps...)

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if _, err := e.agent.Handle(context.Background(), alisher, "покажи траты"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	want := strings.Repeat("user,assistant,tool,assistant,", 10)
	if got := roles(e.stored(t)) + ","; got != want {
		t.Errorf("turns interleaved: %s", got)
	}
}

func TestHistoryContractMemstore(t *testing.T) {
	agenttest.RunHistoryContract(t, func(*testing.T) agent.History {
		return memstore.New(now).History()
	})
}
