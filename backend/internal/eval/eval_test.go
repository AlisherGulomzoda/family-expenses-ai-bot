package eval

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm/llmtest"
)

func moscow(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestShippedSuiteLoads(t *testing.T) {
	s, err := LoadSuite("../../eval/cases.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Cases) < 40 {
		t.Errorf("%d cases, want at least 40", len(s.Cases))
	}
}

func TestLoadSuiteRejectsTypos(t *testing.T) {
	path := t.TempDir() + "/cases.yaml"
	write := func(body string) {
		t.Helper()
		if err := writeFile(path, body); err != nil {
			t.Fatal(err)
		}
	}
	write("today: \"2026-10-16\"\ncases:\n  - id: a\n    input: x\n    expect: { addded: [] }\n")
	if _, err := LoadSuite(path); err == nil || !strings.Contains(err.Error(), "addded") {
		t.Errorf("unknown field: err = %v", err)
	}
	write("today: \"2026-10-16\"\ncases:\n  - id: a\n    input: x\n    expect: { deleted: [nope] }\n")
	if _, err := LoadSuite(path); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("unknown seed key: err = %v", err)
	}
}

func TestReplyAmounts(t *testing.T) {
	got := replyAmounts("Итого 3\u00a0830,50 ₽, из них кофе — 350 ₽ и такси 1 200,5 ₽ (12.10)")
	want := []float64{3830.5, 350, 1200.5, 12.1}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func call(name, args string) llm.ToolCall {
	return llm.ToolCall{ID: "c-" + name, Name: name, Arguments: args}
}

func runFake(t *testing.T, c Case, steps ...llmtest.Step) CaseResult {
	t.Helper()
	suite := &Suite{Today: "2026-10-16", Seed: []SeedExpense{
		{Key: "coffee", Amount: "350", Category: "food", SpentAt: "2026-10-16", Comment: "кофе"},
		{Key: "m_taxi", User: "madina", Amount: "999", Category: "transport", SpentAt: "2026-10-15", Comment: "такси"},
	}}
	r := &Runner{Suite: suite, Model: llmtest.New(steps...), Location: moscow(t)}
	return r.Run(context.Background(), c)
}

func TestRunPassesCorrectBehaviour(t *testing.T) {
	c := Case{ID: "add", Input: "такси 1200 вчера", Expect: Expect{
		Added: []Match{{Amount: "1200", Category: "transport", SpentAt: "2026-10-15"}},
		Tools: []string{"add_expense"},
	}}
	res := runFake(t, c,
		llmtest.Call(call("add_expense", `{"amount":"1200","category":"transport","spent_at":"2026-10-15","comment":"такси"}`)),
		llmtest.Reply("Записал: такси, 1 200 ₽, вчера."),
	)
	if !res.Passed || res.Err != nil {
		t.Fatalf("failures: %v, err: %v", res.Failures, res.Err)
	}
}

func TestRunCatchesWrongBehaviour(t *testing.T) {
	tests := []struct {
		name  string
		c     Case
		steps []llmtest.Step
		want  string // part of a failure message
	}{
		{
			name: "wrong category",
			c:    Case{ID: "a", Input: "такси 1200", Expect: Expect{Added: []Match{{Amount: "1200", Category: "transport"}}}},
			steps: []llmtest.Step{
				llmtest.Call(call("add_expense", `{"amount":"1200","category":"other"}`)),
				llmtest.Reply("Записал."),
			},
			want: "expected a new expense",
		},
		{
			name: "side effect on another expense",
			c:    Case{ID: "b", Input: "сколько я потратил"},
			steps: []llmtest.Step{
				llmtest.Call(call("update_expense", `{"id":1,"amount":"1"}`)),
				llmtest.Reply("350 ₽"),
			},
			want: "coffee: changed unexpectedly",
		},
		{
			name: "guessed instead of asking",
			c:    Case{ID: "c", Input: "купил кофе", Expect: Expect{Asks: true}},
			steps: []llmtest.Step{
				llmtest.Call(call("add_expense", `{"amount":"200","category":"food"}`)),
				llmtest.Reply("Записал кофе за 200 ₽."),
			},
			want: "unexpected new expenses",
		},
		{
			name:  "wrong total",
			c:     Case{ID: "d", Input: "сколько?", Expect: Expect{ReplyAmounts: []string{"350"}, Tools: []string{"summarize_expenses"}}},
			steps: []llmtest.Step{llmtest.Reply("Вы потратили 400 ₽.")},
			want:  "reply does not state 350",
		},
		{
			name:  "leaks another user's numbers",
			c:     Case{ID: "e", Input: "покажи траты Мадины", Expect: Expect{ReplyNot: []string{"999"}}},
			steps: []llmtest.Step{llmtest.Reply("У Мадины такси за 999 ₽.")},
			want:  `must not contain "999"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := runFake(t, tt.c, tt.steps...)
			if res.Passed {
				t.Fatal("wrong behaviour passed")
			}
			if !strings.Contains(strings.Join(res.Failures, "\n"), tt.want) {
				t.Errorf("failures %q do not mention %q", res.Failures, tt.want)
			}
		})
	}
}

func TestReport(t *testing.T) {
	s := Summary{Model: "fake", Runs: 1, Started: time.Now(), Finished: time.Now(), Results: []CaseResult{
		{Case: Case{ID: "ok", Tags: []string{"add"}, Input: "кофе 350"}, Passed: true, Reply: "Записал"},
		{Case: Case{ID: "bad", Tags: []string{"add"}, Input: "такси 1200"}, Failures: []string{"expected a new expense"}, Reply: "Нет | да"},
	}}
	var b strings.Builder
	if err := s.WriteMarkdown(&b); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"**50%** (1 / 2)", "| add | 1 / 2 |", "### bad", "expected a new expense", `Нет \| да`} {
		if !strings.Contains(out, want) {
			t.Errorf("report has no %q:\n%s", want, out)
		}
	}
}

func writeFile(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o600)
}
