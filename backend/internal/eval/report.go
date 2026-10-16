package eval

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"time"
)

// Summary aggregates the results of a run.
type Summary struct {
	Model    string
	Runs     int // runs per case
	Results  []CaseResult
	Started  time.Time
	Finished time.Time
}

func (s Summary) passed() (n int) {
	for _, r := range s.Results {
		if r.Passed {
			n++
		}
	}
	return n
}

// PassRate is the share of passed case runs.
func (s Summary) PassRate() float64 {
	if len(s.Results) == 0 {
		return 0
	}
	return float64(s.passed()) / float64(len(s.Results))
}

type tagStat struct{ passed, total int }

func (s Summary) byTag() (map[string]*tagStat, []string) {
	stats := map[string]*tagStat{}
	for _, r := range s.Results {
		for _, t := range r.Case.Tags {
			st, ok := stats[t]
			if !ok {
				st = &tagStat{}
				stats[t] = st
			}
			st.total++
			if r.Passed {
				st.passed++
			}
		}
	}
	tags := make([]string, 0, len(stats))
	for t := range stats {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return stats, tags
}

func (s Summary) latency() (median, p90 time.Duration) {
	ds := make([]time.Duration, 0, len(s.Results))
	for _, r := range s.Results {
		if r.Err == nil {
			ds = append(ds, r.Duration)
		}
	}
	if len(ds) == 0 {
		return 0, 0
	}
	slices.Sort(ds)
	return ds[len(ds)/2], ds[(len(ds)*9)/10]
}

// WriteMarkdown writes the report.
func (s Summary) WriteMarkdown(w io.Writer) error {
	var b strings.Builder
	p := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }

	median, p90 := s.latency()
	var prompt, completion int
	for _, r := range s.Results {
		prompt += r.Usage.PromptTokens
		completion += r.Usage.CompletionTokens
	}
	n := max(len(s.Results), 1)

	p("# Eval: %s\n\n", s.Model)
	p("_%s, %d %s, %d run(s) per case._\n\n", s.Started.Format("2006-01-02 15:04"), len(s.Results)/max(s.Runs, 1),
		"cases", s.Runs)
	p("| Metric | Value |\n|---|---|\n")
	p("| **Pass rate** | **%.0f%%** (%d / %d) |\n", s.PassRate()*100, s.passed(), len(s.Results))
	p("| Latency per message, median / p90 | %.1f s / %.1f s |\n", median.Seconds(), p90.Seconds())
	p("| Tokens per message, prompt / completion | %d / %d |\n", prompt/n, completion/n)
	p("| Wall time | %s |\n\n", s.Finished.Sub(s.Started).Round(time.Second))

	stats, tags := s.byTag()
	p("## By tag\n\n| Tag | Passed |\n|---|---|\n")
	for _, t := range tags {
		st := stats[t]
		p("| %s | %d / %d |\n", t, st.passed, st.total)
	}

	var failed []CaseResult
	for _, r := range s.Results {
		if !r.Passed {
			failed = append(failed, r)
		}
	}
	p("\n## Failures (%d)\n\n", len(failed))
	if len(failed) == 0 {
		p("None.\n")
	}
	for _, r := range failed {
		p("### %s `%s`\n\n", r.Case.ID, strings.Join(r.Case.Tags, ", "))
		p("> %s\n\n", r.Case.Input)
		if r.Err != nil {
			p("- **error:** %s\n", r.Err)
		}
		for _, f := range r.Failures {
			p("- %s\n", f)
		}
		p("- tools: %s\n", toolCalls(r))
		p("- reply: %q\n\n", r.Reply)
	}

	p("## All cases\n\n| Case | Result | Tools | Reply |\n|---|---|---|---|\n")
	for _, r := range s.Results {
		mark := "✅"
		if !r.Passed {
			mark = "❌"
		}
		p("| %s | %s | %s | %s |\n", r.Case.ID, mark, toolNames(r.Tools), cell(r.Reply))
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func toolCalls(r CaseResult) string {
	if len(r.Tools) == 0 {
		return "none"
	}
	parts := make([]string, len(r.Tools))
	for i, t := range r.Tools {
		parts[i] = fmt.Sprintf("`%s(%s)`", t.Name, t.Arguments)
		if t.Error != "" {
			parts[i] += " → error: " + t.Error
		}
	}
	return strings.Join(parts, ", ")
}

// cell makes a reply fit into one table cell.
func cell(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	if r := []rune(s); len(r) > 120 {
		s = string(r[:120]) + "…"
	}
	return s
}
