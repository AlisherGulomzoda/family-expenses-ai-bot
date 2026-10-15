package expense

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func ptr[T any](v T) *T { return &v }

func fieldsOf(t *testing.T, err error) map[string]string {
	t.Helper()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("error = %v, want *ValidationError", err)
	}
	return ve.Fields
}

func TestValidateDraft(t *testing.T) {
	r := rules{today: date(2026, 10, 16)}

	t.Run("defaults and normalization", func(t *testing.T) {
		d := Draft{Amount: 35000, Category: CategoryFood, Comment: ptr("  кофе  ")}
		if err := r.validateDraft(&d); err != nil {
			t.Fatalf("validateDraft: %v", err)
		}
		if !d.SpentAt.Equal(r.today) {
			t.Errorf("SpentAt = %v, want today", d.SpentAt)
		}
		if d.Comment == nil || *d.Comment != "кофе" {
			t.Errorf("Comment = %v, want trimmed", d.Comment)
		}
	})

	t.Run("blank comment becomes nil", func(t *testing.T) {
		d := Draft{Amount: 100, Category: CategoryOther, Comment: ptr("   ")}
		if err := r.validateDraft(&d); err != nil {
			t.Fatalf("validateDraft: %v", err)
		}
		if d.Comment != nil {
			t.Errorf("Comment = %q, want nil", *d.Comment)
		}
	})

	t.Run("time of day is dropped", func(t *testing.T) {
		d := Draft{Amount: 100, Category: CategoryOther, SpentAt: time.Date(2026, 10, 15, 23, 30, 0, 0, time.UTC)}
		if err := r.validateDraft(&d); err != nil {
			t.Fatalf("validateDraft: %v", err)
		}
		if !d.SpentAt.Equal(date(2026, 10, 15)) {
			t.Errorf("SpentAt = %v", d.SpentAt)
		}
	})

	tests := []struct {
		name  string
		draft Draft
		field string
	}{
		{"zero amount", Draft{Amount: 0, Category: CategoryFood}, "amount"},
		{"too large amount", Draft{Amount: MaxMoney + 1, Category: CategoryFood}, "amount"},
		{"unknown category", Draft{Amount: 100, Category: "fun"}, "category"},
		{"empty category", Draft{Amount: 100}, "category"},
		{"day after tomorrow", Draft{Amount: 100, Category: CategoryFood, SpentAt: date(2026, 10, 18)}, "spent_at"},
		{"before 2000", Draft{Amount: 100, Category: CategoryFood, SpentAt: date(1999, 12, 31)}, "spent_at"},
		{"long comment", Draft{Amount: 100, Category: CategoryFood, Comment: ptr(strings.Repeat("ж", 501))}, "comment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.draft
			fields := fieldsOf(t, r.validateDraft(&d))
			if _, ok := fields[tt.field]; !ok || len(fields) != 1 {
				t.Errorf("fields = %v, want only %q", fields, tt.field)
			}
		})
	}

	t.Run("tomorrow and 500 runes are allowed", func(t *testing.T) {
		d := Draft{Amount: 100, Category: CategoryFood, SpentAt: date(2026, 10, 17), Comment: ptr(strings.Repeat("ж", 500))}
		if err := r.validateDraft(&d); err != nil {
			t.Fatalf("validateDraft: %v", err)
		}
	})
}

func TestValidatePatch(t *testing.T) {
	r := rules{today: date(2026, 10, 16)}

	p := Patch{}
	if err := r.validatePatch(&p); err != nil || !p.IsEmpty() {
		t.Fatalf("empty patch: err=%v empty=%v", err, p.IsEmpty())
	}

	p = Patch{Comment: ptr("  ")}
	if err := r.validatePatch(&p); err != nil {
		t.Fatalf("validatePatch: %v", err)
	}
	if p.Comment == nil || *p.Comment != "" {
		t.Errorf("blank comment should become \"\" (clear), got %v", p.Comment)
	}

	p = Patch{Amount: ptr(Money(0)), Category: ptr(Category("fun")), SpentAt: ptr(date(2030, 1, 1))}
	fields := fieldsOf(t, r.validatePatch(&p))
	for _, f := range []string{"amount", "category", "spent_at"} {
		if _, ok := fields[f]; !ok {
			t.Errorf("missing field %q in %v", f, fields)
		}
	}
}

func TestFilterNormalize(t *testing.T) {
	f := Filter{}
	if err := f.normalize(); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if f.Sort != SortSpentAt || f.Order != OrderDesc || f.Limit != DefaultLimit {
		t.Errorf("defaults = %+v", f)
	}

	bad := Filter{
		Sort: "id; DROP TABLE expenses", Order: "sideways", Limit: 101, Offset: -1,
		Category: ptr(Category("fun")), From: ptr(date(2026, 10, 2)), To: ptr(date(2026, 10, 1)),
	}
	fields := fieldsOf(t, bad.normalize())
	for _, k := range []string{"sort", "order", "limit", "offset", "category", "from"} {
		if _, ok := fields[k]; !ok {
			t.Errorf("missing field %q in %v", k, fields)
		}
	}

	same := Filter{From: ptr(date(2026, 10, 1)), To: ptr(date(2026, 10, 1))}
	if err := same.normalize(); err != nil {
		t.Errorf("one-day period: %v", err)
	}
}

func TestCategories(t *testing.T) {
	cats := Categories()
	if len(cats) != 7 || cats[0].Code != CategoryFood || cats[6].Code != CategoryOther {
		t.Fatalf("Categories() = %v", cats)
	}
	cats[0].Name = "changed"
	if Categories()[0].Name != "Еда" {
		t.Error("Categories() must return a copy")
	}
	if _, err := ParseCategory("entertainment"); err != nil {
		t.Error(err)
	}
	if _, err := ParseCategory("fun"); err == nil {
		t.Error("ParseCategory(fun) succeeded")
	}
	if CategoryHousing.Name() != "Жильё" {
		t.Errorf("Name() = %q", CategoryHousing.Name())
	}
}
