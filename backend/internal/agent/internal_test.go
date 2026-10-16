package agent

import (
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
)

var update = flag.Bool("update", false, "rewrite golden files")

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestSystemPromptGolden(t *testing.T) {
	got, err := systemPrompt("Алишер", "Europe/Moscow", date(2026, 10, 16))
	if err != nil {
		t.Fatal(err)
	}
	const golden = "testdata/prompt.golden"
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run with -update to create it)", err)
	}
	if got != string(want) {
		t.Errorf("prompt differs from %s (run with -update if the change is intended):\n%s", golden, got)
	}
}

func TestSystemPromptPeriods(t *testing.T) {
	tests := []struct {
		name  string
		today time.Time
		want  []string
	}{
		{"friday mid-month", date(2026, 10, 16), []string{
			"Сегодня пятница, 2026-10-16",
			"вчера: 2026-10-15",
			"эта неделя: 2026-10-12 — 2026-10-18",
			"этот месяц: 2026-10-01 — 2026-10-31",
			"прошлый месяц: 2026-09-01 — 2026-09-30",
		}},
		{"sunday ends the week", date(2026, 10, 18), []string{
			"Сегодня воскресенье",
			"эта неделя: 2026-10-12 — 2026-10-18",
		}},
		{"monday starts the week", date(2026, 10, 19), []string{
			"эта неделя: 2026-10-19 — 2026-10-25",
		}},
		{"january: previous month is in the previous year", date(2026, 1, 1), []string{
			"вчера: 2025-12-31",
			"прошлый месяц: 2025-12-01 — 2025-12-31",
		}},
		{"february of a leap year", date(2028, 3, 10), []string{
			"прошлый месяц: 2028-02-01 — 2028-02-29",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := systemPrompt("Мадина", "Europe/Moscow", tt.today)
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range tt.want {
				if !strings.Contains(p, w) {
					t.Errorf("prompt has no %q", w)
				}
			}
			if !strings.Contains(p, "Мадина") || !strings.Contains(p, "Europe/Moscow") {
				t.Error("prompt has no user name or time zone")
			}
		})
	}
}

func TestTrimToTurn(t *testing.T) {
	u := llm.Message{Role: llm.RoleUser}
	call := llm.Message{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "1"}}}
	result := llm.Message{Role: llm.RoleTool, ToolCallID: "1"}
	answer := llm.Message{Role: llm.RoleAssistant, Content: "ok"}

	tests := []struct {
		name string
		in   []llm.Message
		want int // length of the result
	}{
		{"empty", nil, 0},
		{"starts with a user message", []llm.Message{u, call, result, answer}, 4},
		{"cut after a tool call", []llm.Message{result, answer, u, answer}, 2},
		{"cut inside a turn", []llm.Message{call, result, answer, u, call, result, answer}, 4},
		{"no user message", []llm.Message{result, answer}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimToTurn(tt.in)
			if len(got) != tt.want {
				t.Fatalf("len = %d, want %d", len(got), tt.want)
			}
			if len(got) > 0 && got[0].Role != llm.RoleUser {
				t.Errorf("window starts with %s", got[0].Role)
			}
		})
	}
}
