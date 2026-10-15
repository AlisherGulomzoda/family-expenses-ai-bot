// Package expense is the domain of family expenses: the entity, its
// validation rules and the service that enforces who may touch what.
//
// The package knows nothing about HTTP, Telegram, LLMs or SQL.
package expense

import (
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Expense is one spending of a family member.
type Expense struct {
	ID       int64
	UserID   int64
	UserName string // filled by the repository from the users table
	Amount   Money
	Category Category
	// SpentAt is a calendar date stored as midnight UTC.
	SpentAt   time.Time
	Comment   *string
	Source    Source
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Limits of field values.
const (
	MaxCommentLength = 500
	DefaultLimit     = 20
	MaxLimit         = 100
)

// MinSpentAt is the earliest accepted expense date.
var MinSpentAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// ErrNotFound is returned when an expense does not exist or belongs to
// another user. The two cases are deliberately indistinguishable.
var ErrNotFound = errors.New("expense not found")

// ValidationError lists invalid fields with messages safe to show to people.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	keys := make([]string, 0, len(e.Fields))
	for k := range e.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+": "+e.Fields[k])
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

// fieldErrors collects field messages and turns into an error if any.
type fieldErrors map[string]string

func (f fieldErrors) add(field, msg string) {
	if _, ok := f[field]; !ok {
		f[field] = msg
	}
}

func (f fieldErrors) err() error {
	if len(f) == 0 {
		return nil
	}
	return &ValidationError{Fields: f}
}

// Draft is a new expense without an owner. The bot creates expenses from
// drafts, so it has no way to name another user.
type Draft struct {
	Amount   Money
	Category Category
	// SpentAt zero value means "today".
	SpentAt time.Time
	Comment *string
}

// CreateInput is a new expense for any family member (web panel).
type CreateInput struct {
	UserID int64
	Draft
}

// Patch changes some fields of an expense; nil means "keep".
// Comment pointing to an empty string clears the comment.
type Patch struct {
	Amount   *Money
	Category *Category
	SpentAt  *time.Time
	Comment  *string
}

// UpdateInput is a change of any expense, possibly moving it to another
// family member (web panel). The bot uses Patch, which cannot do that.
type UpdateInput struct {
	UserID *int64
	Patch
}

// NewExpense is what the service passes to the repository to create a row.
type NewExpense struct {
	UserID   int64
	Amount   Money
	Category Category
	SpentAt  time.Time
	Comment  *string
	Source   Source
}

// rules validates inputs; it holds "today" for the date checks.
type rules struct {
	today time.Time
}

func checkAmount(errs fieldErrors, m Money) {
	switch {
	case m <= 0:
		errs.add("amount", "Укажите сумму больше нуля")
	case m > MaxMoney:
		errs.add("amount", "Сумма не может быть больше 100 000 000 ₽")
	}
}

func checkCategory(errs fieldErrors, c Category) {
	if !c.Valid() {
		errs.add("category", "Выберите категорию")
	}
}

// checkSpentAt allows tomorrow: a family member in another time zone
// may already be living in it.
func (r rules) checkSpentAt(errs fieldErrors, d time.Time) {
	switch {
	case d.Before(MinSpentAt):
		errs.add("spent_at", "Дата не может быть раньше 01.01.2000")
	case d.After(r.today.AddDate(0, 0, 1)):
		errs.add("spent_at", "Дата не может быть в будущем")
	}
}

func checkComment(errs fieldErrors, c *string) {
	if c != nil && utf8.RuneCountInString(*c) > MaxCommentLength {
		errs.add("comment", "Комментарий не длиннее 500 символов")
	}
}

// normalizeComment trims spaces; an empty result means "no comment".
func normalizeComment(c *string) *string {
	if c == nil {
		return nil
	}
	s := strings.TrimSpace(*c)
	if s == "" {
		return nil
	}
	return &s
}

// validateDraft normalizes d in place and checks it.
func (r rules) validateDraft(d *Draft) error {
	errs := fieldErrors{}
	if d.SpentAt.IsZero() {
		d.SpentAt = r.today
	}
	d.SpentAt = dateOnly(d.SpentAt)
	d.Comment = normalizeComment(d.Comment)

	checkAmount(errs, d.Amount)
	checkCategory(errs, d.Category)
	r.checkSpentAt(errs, d.SpentAt)
	checkComment(errs, d.Comment)
	return errs.err()
}

// validatePatch normalizes p in place and checks the fields it changes.
// A comment cleared to "" stays "" so the repository knows to set NULL.
func (r rules) validatePatch(p *Patch) error {
	errs := fieldErrors{}
	if p.Amount != nil {
		checkAmount(errs, *p.Amount)
	}
	if p.Category != nil {
		checkCategory(errs, *p.Category)
	}
	if p.SpentAt != nil {
		d := dateOnly(*p.SpentAt)
		p.SpentAt = &d
		r.checkSpentAt(errs, d)
	}
	if p.Comment != nil {
		s := strings.TrimSpace(*p.Comment)
		p.Comment = &s
		checkComment(errs, p.Comment)
	}
	return errs.err()
}

// IsEmpty reports whether p changes nothing.
func (p Patch) IsEmpty() bool {
	return p.Amount == nil && p.Category == nil && p.SpentAt == nil && p.Comment == nil
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
