// Package eval measures the expense agent against a real LLM: each case is a
// user message on a known set of expenses, and the result is judged by the
// data it leaves behind (and the reply), not by the exact tool arguments —
// a model may reach the right result in more than one way.
package eval

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

// Family members of every case. Alisher talks to the bot; Madina's expenses
// must never be visible to him or changed by him.
const (
	Alisher int64 = 1
	Madina  int64 = 2
)

// Suite is the file with the cases.
type Suite struct {
	// Today is the date the agent believes it is (Europe/Moscow).
	Today string `yaml:"today"`
	// Seed is the default set of expenses; a case may replace it.
	Seed  []SeedExpense `yaml:"seed"`
	Cases []Case        `yaml:"cases"`
}

// SeedExpense is an expense that exists before the case runs.
type SeedExpense struct {
	Key      string `yaml:"key"`  // name used in expectations
	User     string `yaml:"user"` // "alisher" (default) or "madina"
	Amount   string `yaml:"amount"`
	Category string `yaml:"category"`
	SpentAt  string `yaml:"spent_at"`
	Comment  string `yaml:"comment"`
}

// Case is one message and what should come of it.
type Case struct {
	ID   string   `yaml:"id"`
	Tags []string `yaml:"tags"`
	// Seed replaces the suite seed when set (an empty list means no expenses).
	Seed *[]SeedExpense `yaml:"seed"`
	// Before are earlier messages of the same conversation, not judged.
	Before []string `yaml:"before"`
	Input  string   `yaml:"input"`
	Expect Expect   `yaml:"expect"`
}

// Expect lists what must be true after the message. Anything not mentioned
// must stay as it was: no extra expenses, no other changes.
type Expect struct {
	// Added are new expenses of Alisher, matched in any order.
	Added []Match `yaml:"added"`
	// Updated maps seed keys to the fields they must have now.
	Updated map[string]Match `yaml:"updated"`
	// Deleted are seed keys that must be gone.
	Deleted []string `yaml:"deleted"`
	// Tools must have been called (e.g. summarize_expenses for "сколько").
	Tools []string `yaml:"tools"`
	// ReplyAmounts are ruble amounts the reply must state.
	ReplyAmounts []string `yaml:"reply_amounts"`
	// ReplyAny: the reply contains at least one of these (case-insensitive).
	ReplyAny []string `yaml:"reply_any"`
	// ReplyAll: the reply contains every one of these (case-insensitive).
	ReplyAll []string `yaml:"reply_all"`
	// ReplyNot: the reply contains none of these.
	ReplyNot []string `yaml:"reply_not"`
	// Asks: the reply is a clarifying question.
	Asks bool `yaml:"asks"`
}

// Match describes an expense; empty fields are not checked.
type Match struct {
	Amount string `yaml:"amount"`
	// Category or Categories (any of them).
	Category        string   `yaml:"category"`
	Categories      []string `yaml:"categories"`
	SpentAt         string   `yaml:"spent_at"`
	CommentContains string   `yaml:"comment_contains"`
}

// LoadSuite reads and validates a cases file.
func LoadSuite(path string) (*Suite, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path of the cases file from a flag of a local dev tool
	if err != nil {
		return nil, err
	}
	var s Suite
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true) // a typo in a field name must not silently disable a check
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := s.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &s, nil
}

func (s *Suite) validate() error {
	if _, err := time.Parse(time.DateOnly, s.Today); err != nil {
		return fmt.Errorf("today: %w", err)
	}
	ids := map[string]bool{}
	for _, c := range s.Cases {
		if c.ID == "" || c.Input == "" {
			return fmt.Errorf("case %q: id and input are required", c.ID)
		}
		if ids[c.ID] {
			return fmt.Errorf("duplicate case id %q", c.ID)
		}
		ids[c.ID] = true
		keys := map[string]bool{}
		for _, e := range s.seedOf(c) {
			keys[e.Key] = true
		}
		for k := range c.Expect.Updated {
			if !keys[k] {
				return fmt.Errorf("case %q: updated key %q is not in the seed", c.ID, k)
			}
		}
		for _, k := range c.Expect.Deleted {
			if !keys[k] {
				return fmt.Errorf("case %q: deleted key %q is not in the seed", c.ID, k)
			}
		}
	}
	return nil
}

func (s *Suite) seedOf(c Case) []SeedExpense {
	if c.Seed != nil {
		return *c.Seed
	}
	return s.Seed
}
