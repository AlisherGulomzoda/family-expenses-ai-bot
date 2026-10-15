package memstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/memstore"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

func date(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }

func ptr[T any](v T) *T { return &v }

func seed(t *testing.T) memstore.Expenses {
	t.Helper()
	ctx := context.Background()
	s := memstore.New(clock.Fixed(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)))
	if _, err := s.Users().Upsert(ctx, user.User{TelegramID: 1, Name: "Алишер"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Users().Upsert(ctx, user.User{TelegramID: 2, Name: "Мадина"}); err != nil {
		t.Fatal(err)
	}
	repo := s.Expenses()
	rows := []expense.NewExpense{
		{UserID: 1, Amount: 35000, Category: expense.CategoryFood, SpentAt: date(15), Comment: ptr("Кофе")},
		{UserID: 2, Amount: 120000, Category: expense.CategoryTransport, SpentAt: date(14), Comment: ptr("такси")},
		{UserID: 1, Amount: 348050, Category: expense.CategoryFood, SpentAt: date(14), Comment: ptr("продукты")},
		{UserID: 2, Amount: 120000, Category: expense.CategoryHealth, SpentAt: date(1)},
	}
	for _, r := range rows {
		r.Source = expense.SourceBot
		if _, err := repo.Create(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

func ids(items []expense.Expense) []int64 {
	out := make([]int64, len(items))
	for i, e := range items {
		out[i] = e.ID
	}
	return out
}

func TestListSortAndPage(t *testing.T) {
	repo := seed(t)
	ctx := context.Background()

	tests := []struct {
		name string
		f    expense.Filter
		want []int64
	}{
		{"spent_at desc, ties by id desc", expense.Filter{Sort: expense.SortSpentAt, Order: expense.OrderDesc}, []int64{1, 3, 2, 4}},
		{"spent_at asc", expense.Filter{Sort: expense.SortSpentAt, Order: expense.OrderAsc}, []int64{4, 2, 3, 1}},
		{"amount desc", expense.Filter{Sort: expense.SortAmount, Order: expense.OrderDesc}, []int64{3, 4, 2, 1}},
		{"page 2", expense.Filter{Sort: expense.SortSpentAt, Order: expense.OrderDesc, Limit: 2, Offset: 2}, []int64{2, 4}},
		{"offset past end", expense.Filter{Sort: expense.SortSpentAt, Order: expense.OrderDesc, Offset: 10}, []int64{}},
		{"query is case-insensitive", expense.Filter{Query: "КОФ", Order: expense.OrderDesc}, []int64{1}},
		{"period inclusive", expense.Filter{From: ptr(date(14)), To: ptr(date(14)), Order: expense.OrderDesc}, []int64{3, 2}},
		{"user and category", expense.Filter{UserID: ptr(int64(1)), Category: ptr(expense.CategoryFood), Order: expense.OrderDesc}, []int64{1, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := repo.List(ctx, tt.f)
			if err != nil {
				t.Fatal(err)
			}
			got := ids(items)
			if len(got) != len(tt.want) {
				t.Fatalf("ids = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("ids = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSummary(t *testing.T) {
	repo := seed(t)
	sum, err := repo.Summary(context.Background(), expense.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if sum.Total != 623050 || sum.Count != 4 {
		t.Fatalf("summary = %+v", sum)
	}
	// food 3830.50 first; transport and health tie at 1200 and keep display order.
	want := []expense.Category{expense.CategoryFood, expense.CategoryTransport, expense.CategoryHealth}
	if len(sum.ByCategory) != len(want) {
		t.Fatalf("by category = %+v", sum.ByCategory)
	}
	for i, c := range want {
		if sum.ByCategory[i].Category != c {
			t.Fatalf("by category = %+v, want order %v", sum.ByCategory, want)
		}
	}
	if sum.ByCategory[0].Total != 383050 || sum.ByCategory[0].Count != 2 {
		t.Errorf("food = %+v", sum.ByCategory[0])
	}

	empty, err := repo.Summary(context.Background(), expense.Filter{UserID: ptr(int64(99))})
	if err != nil || empty.Total != 0 || empty.Count != 0 || empty.ByCategory == nil {
		t.Errorf("empty summary = %+v, %v (by_category must be [] not nil)", empty, err)
	}
}

func TestUpdateClearsComment(t *testing.T) {
	repo := seed(t)
	got, err := repo.Update(context.Background(), 1, nil, expense.UpdateInput{Patch: expense.Patch{Comment: ptr("")}})
	if err != nil || got.Comment != nil {
		t.Fatalf("Update = %+v, %v", got, err)
	}
}

func TestReturnedValuesAreCopies(t *testing.T) {
	repo := seed(t)
	ctx := context.Background()
	e, err := repo.Get(ctx, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	*e.Comment = "changed"
	again, _ := repo.Get(ctx, 1, nil)
	if *again.Comment != "Кофе" {
		t.Error("store data was modified through a returned pointer")
	}
}
