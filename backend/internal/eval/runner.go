package eval

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent/tools"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/memstore"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

// CaseResult is the outcome of one run of one case.
type CaseResult struct {
	Case     Case
	Passed   bool
	Failures []string
	Reply    string
	Tools    []agent.ToolTrace
	Steps    int
	Usage    llm.Usage
	Duration time.Duration
	// Err is a failure to run the case at all (e.g. the LLM is unreachable).
	Err error
}

// Runner runs cases against one model.
type Runner struct {
	Suite    *Suite
	Model    llm.Client
	Location *time.Location
}

var family = []user.User{
	{TelegramID: Alisher, Name: "Алишер"},
	{TelegramID: Madina, Name: "Мадина"},
}

// Run executes one case on a fresh in-memory store.
func (r *Runner) Run(ctx context.Context, c Case) CaseResult {
	res := CaseResult{Case: c}

	today, _ := time.ParseInLocation(time.DateOnly, r.Suite.Today, r.Location) // validated on load
	clk := clock.Fixed(today.Add(10 * time.Hour))                              // 10:00 local time
	store := memstore.New(clk)
	for _, u := range family {
		if _, err := store.Users().Upsert(ctx, u); err != nil {
			res.Err = err
			return res
		}
	}
	svc := expense.NewService(store.Expenses(), clk, r.Location)

	seed := r.Suite.seedOf(c)
	ids, err := plantSeed(ctx, svc, seed)
	if err != nil {
		res.Err = fmt.Errorf("seed: %w", err)
		return res
	}

	a := agent.New(r.Model, tools.Default(), svc, store.History(),
		agent.Config{HistoryLimit: 20, MaxSteps: 5}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	me := family[0]
	for _, msg := range c.Before {
		if _, err := a.Run(ctx, me, msg); err != nil {
			res.Err = fmt.Errorf("before %q: %w", msg, err)
			return res
		}
	}

	out, err := a.Run(ctx, me, c.Input)
	res.Reply, res.Tools, res.Steps, res.Usage, res.Duration = out.Reply, out.Tools, out.Steps, out.Usage, out.Duration
	if err != nil {
		res.Err = err
		return res
	}

	res.Failures = check(ctx, svc, c, seed, ids, out)
	res.Passed = len(res.Failures) == 0
	return res
}

// plantSeed creates the seed expenses in order and returns their IDs by key.
// The order matters: the last planted expense is "the last one" for the agent.
func plantSeed(ctx context.Context, svc *expense.Service, seed []SeedExpense) (map[string]int64, error) {
	ids := map[string]int64{}
	for _, s := range seed {
		amount, err := expense.ParseRubles(s.Amount)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.Key, err)
		}
		spent, err := time.Parse(time.DateOnly, s.SpentAt)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.Key, err)
		}
		d := expense.Draft{Amount: amount, Category: expense.Category(s.Category), SpentAt: spent}
		if s.Comment != "" {
			comment := s.Comment
			d.Comment = &comment
		}
		owner := Alisher
		if s.User == "madina" {
			owner = Madina
		}
		e, err := svc.ForUser(owner).Create(ctx, d)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.Key, err)
		}
		ids[s.Key] = e.ID
	}
	return ids, nil
}
