// Command eval runs the agent evaluation suite against the LLM from the
// environment (LLM_BASE_URL, LLM_API_KEY, LLM_MODEL) and writes a Markdown report.
//
//	eval [-cases eval/cases.yaml] [-tags add,read] [-case id] [-runs 3] [-parallel 4] [-out eval/reports]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"
	_ "time/tzdata" // the runtime image has no system time zone database

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/config"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/eval"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "eval:", err)
		os.Exit(1)
	}
}

func run() error {
	casesPath := flag.String("cases", "eval/cases.yaml", "cases file")
	tags := flag.String("tags", "", "only cases with any of these comma-separated tags")
	only := flag.String("case", "", "only the case with this id")
	runs := flag.Int("runs", 1, "runs per case (to see how stable the answers are)")
	parallel := flag.Int("parallel", 4, "cases run at the same time")
	outDir := flag.String("out", "eval/reports", "directory for the Markdown report")
	flag.Parse()

	cfg, err := config.Load[config.Eval]()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, "eval")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	suite, err := eval.LoadSuite(*casesPath)
	if err != nil {
		return err
	}
	cases := selectCases(suite.Cases, *tags, *only)
	if len(cases) == 0 {
		return fmt.Errorf("no cases match")
	}

	model := llm.NewOpenAI(llm.OpenAIConfig{
		BaseURL: cfg.LLM.BaseURL, APIKey: cfg.LLM.APIKey, Model: cfg.LLM.Model,
		Temperature: cfg.LLM.Temperature, Timeout: cfg.LLM.Timeout, MaxRetries: 2,
	})
	runner := &eval.Runner{Suite: suite, Model: model, Location: cfg.Timezone.Location}

	summary := eval.Summary{Model: cfg.LLM.Model, Runs: *runs, Started: time.Now()}
	jobs := make(chan eval.Case)
	results := make(chan eval.CaseResult)
	var wg sync.WaitGroup
	for range max(*parallel, 1) {
		wg.Go(func() {
			for c := range jobs {
				results <- runner.Run(ctx, c)
			}
		})
	}
	go func() {
		defer close(jobs)
		for range *runs {
			for _, c := range cases {
				select {
				case jobs <- c:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	total := len(cases) * *runs
	done := 0
	for r := range results {
		done++
		summary.Results = append(summary.Results, r)
		status := "pass"
		if !r.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(os.Stderr, "[%d/%d] %-4s %s (%.1fs)\n", done, total, status, r.Case.ID, r.Duration.Seconds())
	}
	summary.Finished = time.Now()
	// Stable order in the report: as in the cases file.
	order := map[string]int{}
	for i, c := range cases {
		order[c.ID] = i
	}
	slices.SortStableFunc(summary.Results, func(a, b eval.CaseResult) int { return order[a.Case.ID] - order[b.Case.ID] })

	if err := os.MkdirAll(*outDir, 0o750); err != nil {
		return err
	}
	name := fmt.Sprintf("%s-%s.md", slug(cfg.LLM.Model), summary.Started.Format("2006-01-02-1504"))
	path := filepath.Join(*outDir, name)
	f, err := os.Create(path) //nolint:gosec // path from a flag of a local dev tool
	if err != nil {
		return err
	}
	defer f.Close()
	if err := summary.WriteMarkdown(f); err != nil {
		return err
	}
	log.Info("eval finished", "model", cfg.LLM.Model, "pass_rate", fmt.Sprintf("%.0f%%", summary.PassRate()*100),
		"cases", len(cases), "runs", *runs, "report", path)
	fmt.Printf("%s: %.0f%% passed (%d case runs) → %s\n", cfg.LLM.Model, summary.PassRate()*100, total, path)
	return nil
}

func selectCases(all []eval.Case, tags, only string) []eval.Case {
	want := map[string]bool{}
	for _, t := range strings.Split(tags, ",") {
		if t = strings.TrimSpace(t); t != "" {
			want[t] = true
		}
	}
	var out []eval.Case
	for _, c := range all {
		if only != "" && c.ID != only {
			continue
		}
		if len(want) > 0 && !slices.ContainsFunc(c.Tags, func(t string) bool { return want[t] }) {
			continue
		}
		out = append(out, c)
	}
	return out
}

// slug turns a model name into a file name: "qwen2.5:7b" -> "qwen2.5-7b".
func slug(s string) string {
	return strings.Map(func(r rune) rune {
		if r == ':' || r == '/' || r == ' ' {
			return '-'
		}
		return r
	}, s)
}
