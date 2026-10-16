//go:build live

package agent_test

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent/tools"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/memstore"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

// TestLiveTaskExamples plays the examples from the task as one conversation
// against the real LLM from the environment and checks the stored data:
//
//	go test -tags live -run LiveTaskExamples -v ./internal/agent/
//
// It is a smoke test of the whole agent; the systematic measurement is the
// eval suite (cmd/eval).
func TestLiveTaskExamples(t *testing.T) {
	key := os.Getenv("LLM_API_KEY")
	if key == "" {
		t.Skip("LLM_API_KEY is not set")
	}
	ctx := context.Background()
	moscow, _ := time.LoadLocation("Europe/Moscow")
	// Friday 2026-10-16, 10:00 in Moscow.
	now := clock.Fixed(time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC))
	store := memstore.New(now)
	me := user.User{TelegramID: 1, Name: "Алишер"}
	sister := user.User{TelegramID: 2, Name: "Мадина"}
	for _, u := range []user.User{me, sister} {
		if _, err := store.Users().Upsert(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	svc := expense.NewService(store.Expenses(), now, moscow)
	// Someone else's taxi: must never show up or be deleted.
	foreign := "такси"
	if _, err := svc.ForUser(sister.TelegramID).Create(ctx, expense.Draft{Amount: 99900, Category: expense.CategoryTransport, Comment: &foreign}); err != nil {
		t.Fatal(err)
	}

	model := llm.NewOpenAI(llm.OpenAIConfig{
		BaseURL: envOr("LLM_BASE_URL", "https://api.deepseek.com"), APIKey: key,
		Model: envOr("LLM_MODEL", "deepseek-chat"), Temperature: 0.1, Timeout: 90 * time.Second, MaxRetries: 2,
	})
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	a := agent.New(model, tools.Default(), svc, store.History(), agent.Config{HistoryLimit: 20, MaxSteps: 5}, log)
	mine := svc.ForUser(me.TelegramID)

	say := func(text string) string {
		t.Helper()
		reply, err := a.Handle(ctx, me, text)
		if err != nil {
			t.Fatalf("%q: %v", text, err)
		}
		t.Logf("\n> %s\n< %s", text, reply)
		return reply
	}
	all := func() []expense.Expense {
		t.Helper()
		page, err := mine.List(ctx, expense.Filter{Sort: expense.SortCreatedAt, Order: expense.OrderAsc, Limit: 100})
		if err != nil {
			t.Fatal(err)
		}
		return page.Items
	}

	say("кофе 350")
	if got := all(); len(got) != 1 || got[0].Amount != 35000 || got[0].Category != expense.CategoryFood ||
		got[0].SpentAt.Format(time.DateOnly) != "2026-10-16" {
		t.Fatalf("after «кофе 350»: %+v", got)
	}

	say("последнее было не 350, а 450")
	if got := all(); len(got) != 1 || got[0].Amount != 45000 {
		t.Fatalf("after correction: %+v", got)
	}

	say("такси 1200 вчера")
	got := all()
	if len(got) != 2 || got[1].Amount != 120000 || got[1].Category != expense.CategoryTransport ||
		got[1].SpentAt.Format(time.DateOnly) != "2026-10-15" {
		t.Fatalf("after «такси 1200 вчера»: %+v", got)
	}

	if reply := say("сколько потратил на еду в этом месяце"); !strings.Contains(reply, "450") {
		t.Errorf("food this month should be 450 ₽: %q", reply)
	}

	reply := say("покажи траты за неделю")
	if !strings.Contains(reply, "450") || !strings.Contains(reply, "1") || !strings.Contains(strings.ToLower(reply), "такси") {
		t.Errorf("weekly list should mention coffee 450 and taxi 1 200: %q", reply)
	}
	if strings.Contains(reply, "999") {
		t.Errorf("the weekly list shows someone else's expense: %q", reply)
	}

	say("удали такси")
	if got := all(); len(got) != 1 || got[0].Category != expense.CategoryFood {
		t.Fatalf("after «удали такси»: %+v", got)
	}
	if page, _ := svc.ForUser(sister.TelegramID).List(ctx, expense.Filter{}); page.Total != 1 {
		t.Fatal("someone else's taxi was deleted")
	}

	say("купил продукты")
	if got := all(); len(got) != 1 {
		t.Errorf("an expense without an amount was stored: %+v", got)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
