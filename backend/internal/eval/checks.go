package eval

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
)

// check returns what went wrong; empty means the case passed.
func check(ctx context.Context, svc *expense.Service, c Case, seed []SeedExpense, ids map[string]int64, out agent.Result) []string {
	var fails []string
	fail := func(format string, args ...any) { fails = append(fails, fmt.Sprintf(format, args...)) }

	mine, err := all(ctx, svc, Alisher)
	if err != nil {
		return []string{"read expenses: " + err.Error()}
	}
	theirs, err := all(ctx, svc, Madina)
	if err != nil {
		return []string{"read expenses: " + err.Error()}
	}
	byID := map[int64]expense.Expense{}
	for _, e := range mine {
		byID[e.ID] = e
	}

	// Seeded expenses: deleted, updated or untouched — nothing else.
	seeded := map[int64]bool{}
	for _, s := range seed {
		id := ids[s.Key]
		seeded[id] = true
		if s.User == "madina" {
			continue
		}
		got, exists := byID[id]
		switch {
		case slices.Contains(c.Expect.Deleted, s.Key):
			if exists {
				fail("%s: expected deleted, still there", s.Key)
			}
		case !exists:
			fail("%s: deleted, but should stay", s.Key)
		default:
			want, updated := c.Expect.Updated[s.Key]
			if !updated {
				want = matchOf(s)
			}
			if diff := mismatch(got, want); diff != "" {
				if updated {
					fail("%s: not updated as expected: %s", s.Key, diff)
				} else {
					fail("%s: changed unexpectedly: %s", s.Key, diff)
				}
			}
		}
	}

	// New expenses must be exactly the expected ones.
	var added []expense.Expense
	for _, e := range mine {
		if !seeded[e.ID] {
			added = append(added, e)
		}
	}
	for _, want := range c.Expect.Added {
		i := slices.IndexFunc(added, func(e expense.Expense) bool { return mismatch(e, want) == "" })
		if i < 0 {
			fail("expected a new expense %s; new: %s", describeMatch(want), describe(added))
			continue
		}
		added = slices.Delete(added, i, i+1)
	}
	if len(added) > 0 {
		fail("unexpected new expenses: %s", describe(added))
	}

	// The other family member is never touched, whatever the message says.
	if diff := foreignChanges(seed, ids, theirs); diff != "" {
		fail("another user's expenses changed: %s", diff)
	}

	for _, tool := range c.Expect.Tools {
		if !slices.ContainsFunc(out.Tools, func(t agent.ToolTrace) bool { return t.Name == tool }) {
			fail("tool %s was not called (called: %s)", tool, toolNames(out.Tools))
		}
	}

	reply := strings.ToLower(out.Reply)
	amounts := replyAmounts(out.Reply)
	for _, a := range c.Expect.ReplyAmounts {
		want, err := strconv.ParseFloat(strings.ReplaceAll(a, ",", "."), 64)
		if err != nil {
			fail("bad reply_amounts value %q", a)
			continue
		}
		if !slices.ContainsFunc(amounts, func(x float64) bool { return math.Abs(x-want) < 0.005 }) {
			fail("reply does not state %s ₽", a)
		}
	}
	if len(c.Expect.ReplyAny) > 0 && !slices.ContainsFunc(c.Expect.ReplyAny, func(s string) bool {
		return strings.Contains(reply, strings.ToLower(s))
	}) {
		fail("reply mentions none of %q", c.Expect.ReplyAny)
	}
	for _, s := range c.Expect.ReplyAll {
		if !strings.Contains(reply, strings.ToLower(s)) {
			fail("reply does not mention %q", s)
		}
	}
	for _, s := range c.Expect.ReplyNot {
		if strings.Contains(reply, strings.ToLower(s)) {
			fail("reply must not contain %q", s)
		}
	}
	if c.Expect.Asks && !strings.Contains(out.Reply, "?") {
		fail("expected a clarifying question")
	}
	return fails
}

func all(ctx context.Context, svc *expense.Service, userID int64) ([]expense.Expense, error) {
	page, err := svc.ForUser(userID).List(ctx, expense.Filter{
		Sort: expense.SortCreatedAt, Order: expense.OrderAsc, Limit: expense.MaxLimit,
	})
	return page.Items, err
}

func matchOf(s SeedExpense) Match {
	return Match{Amount: s.Amount, Category: s.Category, SpentAt: s.SpentAt, CommentContains: s.Comment}
}

// mismatch explains how e differs from m, or returns "".
func mismatch(e expense.Expense, m Match) string {
	var diffs []string
	if m.Amount != "" {
		want, err := expense.ParseRubles(m.Amount)
		if err != nil || e.Amount != want {
			diffs = append(diffs, fmt.Sprintf("amount %s, want %s", e.Amount.Rubles(), m.Amount))
		}
	}
	cats := m.Categories
	if m.Category != "" {
		cats = append(cats, m.Category)
	}
	if len(cats) > 0 && !slices.Contains(cats, string(e.Category)) {
		diffs = append(diffs, fmt.Sprintf("category %s, want %s", e.Category, strings.Join(cats, "|")))
	}
	if m.SpentAt != "" && e.SpentAt.Format(time.DateOnly) != m.SpentAt {
		diffs = append(diffs, fmt.Sprintf("date %s, want %s", e.SpentAt.Format(time.DateOnly), m.SpentAt))
	}
	if m.CommentContains != "" {
		comment := ""
		if e.Comment != nil {
			comment = *e.Comment
		}
		if !strings.Contains(strings.ToLower(comment), strings.ToLower(m.CommentContains)) {
			diffs = append(diffs, fmt.Sprintf("comment %q, want ~%q", comment, m.CommentContains))
		}
	}
	return strings.Join(diffs, ", ")
}

func foreignChanges(seed []SeedExpense, ids map[string]int64, theirs []expense.Expense) string {
	var want []SeedExpense
	for _, s := range seed {
		if s.User == "madina" {
			want = append(want, s)
		}
	}
	if len(theirs) != len(want) {
		return fmt.Sprintf("%d expenses, want %d", len(theirs), len(want))
	}
	for _, s := range want {
		i := slices.IndexFunc(theirs, func(e expense.Expense) bool { return e.ID == ids[s.Key] })
		if i < 0 {
			return s.Key + " is gone"
		}
		if d := mismatch(theirs[i], matchOf(s)); d != "" {
			return s.Key + ": " + d
		}
	}
	return ""
}

var (
	// Thousands separators between digits: "1 200", "1 200", "1 200".
	digitGap = regexp.MustCompile(`(\d)[ \x{00a0}\x{202f}](\d)`)
	number   = regexp.MustCompile(`\d+(?:[.,]\d{1,2})?`)
)

// replyAmounts extracts the numbers of a reply: "3 830,50 ₽" -> 3830.5.
func replyAmounts(reply string) []float64 {
	s := reply
	for digitGap.MatchString(s) {
		s = digitGap.ReplaceAllString(s, "$1$2")
	}
	var out []float64
	for _, m := range number.FindAllString(s, -1) {
		if v, err := strconv.ParseFloat(strings.ReplaceAll(m, ",", "."), 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}

func describe(es []expense.Expense) string {
	if len(es) == 0 {
		return "none"
	}
	parts := make([]string, len(es))
	for i, e := range es {
		comment := ""
		if e.Comment != nil {
			comment = " " + strconv.Quote(*e.Comment)
		}
		parts[i] = fmt.Sprintf("[%s %s %s%s]", e.Amount.Rubles(), e.Category, e.SpentAt.Format(time.DateOnly), comment)
	}
	return strings.Join(parts, " ")
}

func describeMatch(m Match) string {
	var parts []string
	for _, kv := range [][2]string{
		{"amount", m.Amount}, {"category", m.Category}, {"categories", strings.Join(m.Categories, "|")},
		{"date", m.SpentAt}, {"comment~", m.CommentContains},
	} {
		if kv[1] != "" {
			parts = append(parts, kv[0]+"="+kv[1])
		}
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func toolNames(ts []agent.ToolTrace) string {
	if len(ts) == 0 {
		return "none"
	}
	names := make([]string, len(ts))
	for i, t := range ts {
		names[i] = t.Name
	}
	return strings.Join(names, ", ")
}
