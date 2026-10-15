// Package expensetest holds the behaviour every expense and user repository
// must share. It runs against the in-memory store and against PostgreSQL,
// so tests and the agent evaluation (in-memory) behave like production.
package expensetest

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

// Repos is a fresh, empty pair of repositories.
type Repos struct {
	Expenses expense.Repository
	Users    user.Repository
}

// Family member IDs used by the suite.
const (
	Alisher  int64 = 100000001
	Madina   int64 = 100000002
	Stranger int64 = 999
)

func ptr[T any](v T) *T { return &v }

func day(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }

// RunRepositoryContract runs the shared repository tests. newRepos must
// return empty repositories for every call.
func RunRepositoryContract(t *testing.T, newRepos func(t *testing.T) Repos) {
	t.Run("users", func(t *testing.T) { testUsers(t, newRepos(t)) })
	t.Run("crud", func(t *testing.T) { testCRUD(t, newRepos(t)) })
	t.Run("owner isolation", func(t *testing.T) { testOwnerIsolation(t, newRepos(t)) })
	t.Run("unknown user", func(t *testing.T) { testUnknownUser(t, newRepos(t)) })
	t.Run("filters", func(t *testing.T) { testFilters(t, newRepos(t)) })
	t.Run("sort and pagination", func(t *testing.T) { testSortAndPagination(t, newRepos(t)) })
	t.Run("summary", func(t *testing.T) { testSummary(t, newRepos(t)) })
}

func addUsers(t *testing.T, r Repos) {
	t.Helper()
	ctx := context.Background()
	for _, u := range []user.User{
		{TelegramID: Madina, Name: "Мадина", Username: ptr("madina")},
		{TelegramID: Alisher, Name: "Алишер"},
	} {
		if _, err := r.Users.Upsert(ctx, u); err != nil {
			t.Fatalf("Upsert(%d): %v", u.TelegramID, err)
		}
	}
}

// seed creates four expenses and returns their IDs in creation order:
// Alisher coffee (15th), Madina taxi (14th), Alisher groceries (14th), Madina pharmacy (1st).
func seed(t *testing.T, r Repos) []int64 {
	t.Helper()
	addUsers(t, r)
	rows := []expense.NewExpense{
		{UserID: Alisher, Amount: 35000, Category: expense.CategoryFood, SpentAt: day(15), Comment: ptr("Кофе с коллегами"), Source: expense.SourceBot},
		{UserID: Madina, Amount: 120000, Category: expense.CategoryTransport, SpentAt: day(14), Comment: ptr("такси"), Source: expense.SourceBot},
		{UserID: Alisher, Amount: 348050, Category: expense.CategoryFood, SpentAt: day(14), Comment: ptr("продукты 100%"), Source: expense.SourceWeb},
		{UserID: Madina, Amount: 120000, Category: expense.CategoryHealth, SpentAt: day(1), Source: expense.SourceWeb},
	}
	ids := make([]int64, len(rows))
	for i, row := range rows {
		e, err := r.Expenses.Create(context.Background(), row)
		if err != nil {
			t.Fatalf("Create #%d: %v", i, err)
		}
		ids[i] = e.ID
	}
	return ids
}

func testUsers(t *testing.T, r Repos) {
	ctx := context.Background()
	addUsers(t, r)
	got, err := r.Users.Upsert(ctx, user.User{TelegramID: Alisher, Name: "Алишер Г.", Username: ptr("alisher")})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Алишер Г." || got.Username == nil || *got.Username != "alisher" {
		t.Errorf("Upsert existing = %+v", got)
	}
	users, err := r.Users.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].TelegramID != Alisher || users[1].TelegramID != Madina {
		t.Errorf("List = %+v, want ordered by name", users)
	}
	if users[1].Username == nil || *users[1].Username != "madina" {
		t.Errorf("Madina's username = %v", users[1].Username)
	}
}

func testCRUD(t *testing.T, r Repos) {
	ctx := context.Background()
	addUsers(t, r)

	created, err := r.Expenses.Create(ctx, expense.NewExpense{
		UserID: Alisher, Amount: 35050, Category: expense.CategoryFood,
		SpentAt: day(15), Comment: ptr("кофе"), Source: expense.SourceBot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID <= 0 || created.UserName != "Алишер" || created.Amount != 35050 ||
		created.Category != expense.CategoryFood || !created.SpentAt.Equal(day(15)) ||
		created.Comment == nil || *created.Comment != "кофе" || created.Source != expense.SourceBot ||
		created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("Create = %+v", created)
	}

	got, err := r.Expenses.Get(ctx, created.ID, nil)
	if err != nil || got.ID != created.ID || got.UserName != "Алишер" {
		t.Fatalf("Get = %+v, %v", got, err)
	}

	// Partial update keeps untouched fields.
	upd, err := r.Expenses.Update(ctx, created.ID, nil, expense.UpdateInput{Patch: expense.Patch{Amount: ptr(expense.Money(45000))}})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Amount != 45000 || upd.Category != expense.CategoryFood || upd.Comment == nil || *upd.Comment != "кофе" ||
		upd.Source != expense.SourceBot || upd.UpdatedAt.Before(created.UpdatedAt) {
		t.Errorf("Update amount = %+v", upd)
	}

	// Every field, a cleared comment and a new owner.
	upd, err = r.Expenses.Update(ctx, created.ID, nil, expense.UpdateInput{
		UserID: ptr(Madina),
		Patch: expense.Patch{
			Category: ptr(expense.CategoryEntertainment),
			SpentAt:  ptr(day(10)),
			Comment:  ptr(""),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if upd.UserID != Madina || upd.UserName != "Мадина" || upd.Category != expense.CategoryEntertainment ||
		!upd.SpentAt.Equal(day(10)) || upd.Comment != nil {
		t.Errorf("Update all = %+v", upd)
	}

	deleted, err := r.Expenses.Delete(ctx, created.ID, nil)
	if err != nil || deleted.ID != created.ID || deleted.UserName != "Мадина" {
		t.Fatalf("Delete = %+v, %v", deleted, err)
	}
	if _, err := r.Expenses.Get(ctx, created.ID, nil); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Get deleted: %v", err)
	}
	if _, err := r.Expenses.Delete(ctx, created.ID, nil); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Delete twice: %v", err)
	}
	if _, err := r.Expenses.Update(ctx, created.ID, nil, expense.UpdateInput{Patch: expense.Patch{Amount: ptr(expense.Money(1))}}); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Update deleted: %v", err)
	}
}

func testOwnerIsolation(t *testing.T, r Repos) {
	ctx := context.Background()
	ids := seed(t, r)
	taxi := ids[1] // Madina's

	if _, err := r.Expenses.Get(ctx, taxi, ptr(Alisher)); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Get foreign: %v", err)
	}
	if _, err := r.Expenses.Update(ctx, taxi, ptr(Alisher), expense.UpdateInput{Patch: expense.Patch{Amount: ptr(expense.Money(1))}}); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Update foreign: %v", err)
	}
	if _, err := r.Expenses.Delete(ctx, taxi, ptr(Alisher)); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Delete foreign: %v", err)
	}

	got, err := r.Expenses.Get(ctx, taxi, ptr(Madina))
	if err != nil || got.Amount != 120000 || got.Comment == nil || *got.Comment != "такси" {
		t.Fatalf("foreign attempts changed Madina's expense: %+v, %v", got, err)
	}
	if _, err := r.Expenses.Update(ctx, taxi, ptr(Madina), expense.UpdateInput{Patch: expense.Patch{Amount: ptr(expense.Money(130000))}}); err != nil {
		t.Errorf("Update own: %v", err)
	}
	if _, err := r.Expenses.Delete(ctx, taxi, ptr(Madina)); err != nil {
		t.Errorf("Delete own: %v", err)
	}
}

func testUnknownUser(t *testing.T, r Repos) {
	ctx := context.Background()
	ids := seed(t, r)

	wantUserIDError := func(name string, err error) {
		t.Helper()
		var ve *expense.ValidationError
		if !errors.As(err, &ve) || ve.Fields["user_id"] == "" {
			t.Errorf("%s: err = %v, want ValidationError on user_id", name, err)
		}
	}

	_, err := r.Expenses.Create(ctx, expense.NewExpense{UserID: Stranger, Amount: 100, Category: expense.CategoryFood, SpentAt: day(1), Source: expense.SourceWeb})
	wantUserIDError("Create", err)

	_, err = r.Expenses.Update(ctx, ids[0], nil, expense.UpdateInput{UserID: ptr(Stranger)})
	wantUserIDError("Update existing", err)

	_, err = r.Expenses.Update(ctx, 1_000_000, nil, expense.UpdateInput{UserID: ptr(Stranger)})
	wantUserIDError("Update missing", err)

	if _, err := r.Expenses.Update(ctx, 1_000_000, nil, expense.UpdateInput{UserID: ptr(Madina)}); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Update missing with known user: %v", err)
	}

	got, err := r.Expenses.Get(ctx, ids[0], nil)
	if err != nil || got.UserID != Alisher {
		t.Errorf("failed update changed the owner: %+v, %v", got, err)
	}
}

func listIDs(t *testing.T, r Repos, f expense.Filter) []int64 {
	t.Helper()
	if f.Sort == "" {
		f.Sort = expense.SortSpentAt
	}
	if f.Order == "" {
		f.Order = expense.OrderDesc
	}
	if f.Limit == 0 {
		f.Limit = expense.DefaultLimit
	}
	items, err := r.Expenses.List(context.Background(), f)
	if err != nil {
		t.Fatalf("List(%+v): %v", f, err)
	}
	n, err := r.Expenses.Count(context.Background(), f)
	if err != nil {
		t.Fatalf("Count(%+v): %v", f, err)
	}
	out := make([]int64, len(items))
	for i, e := range items {
		out[i] = e.ID
		if e.UserName == "" {
			t.Errorf("List item %d has no user name", e.ID)
		}
	}
	if f.Offset == 0 && n < len(out) {
		t.Errorf("Count = %d < listed %d", n, len(out))
	}
	return out
}

func pick(ids []int64, idx ...int) []int64 {
	out := make([]int64, len(idx))
	for i, j := range idx {
		out[i] = ids[j]
	}
	return out
}

func testFilters(t *testing.T, r Repos) {
	ids := seed(t, r)
	tests := []struct {
		name string
		f    expense.Filter
		want []int // indexes into ids, in the expected order
	}{
		{"no filter", expense.Filter{}, []int{0, 2, 1, 3}},
		{"user", expense.Filter{UserID: ptr(Madina)}, []int{1, 3}},
		{"category", expense.Filter{Category: ptr(expense.CategoryFood)}, []int{0, 2}},
		{"period is inclusive", expense.Filter{From: ptr(day(14)), To: ptr(day(15))}, []int{0, 2, 1}},
		{"one day", expense.Filter{From: ptr(day(14)), To: ptr(day(14))}, []int{2, 1}},
		{"from only", expense.Filter{From: ptr(day(15))}, []int{0}},
		{"to only", expense.Filter{To: ptr(day(1))}, []int{3}},
		{"query ignores case (Cyrillic)", expense.Filter{Query: "КОФЕ"}, []int{0}},
		{"query matches a substring", expense.Filter{Query: "такс"}, []int{1}},
		{"query percent is literal", expense.Filter{Query: "100%"}, []int{2}},
		{"query underscore is literal", expense.Filter{Query: "_"}, []int{}},
		{"combined", expense.Filter{UserID: ptr(Alisher), Category: ptr(expense.CategoryFood), From: ptr(day(15))}, []int{0}},
		{"nothing", expense.Filter{UserID: ptr(Stranger)}, []int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listIDs(t, r, tt.f)
			if want := pick(ids, tt.want...); !slices.Equal(got, want) {
				t.Errorf("ids = %v, want %v", got, want)
			}
			n, err := r.Expenses.Count(context.Background(), withDefaults(tt.f))
			if err != nil || n != len(tt.want) {
				t.Errorf("Count = %d, %v; want %d", n, err, len(tt.want))
			}
		})
	}
}

func withDefaults(f expense.Filter) expense.Filter {
	if f.Sort == "" {
		f.Sort = expense.SortSpentAt
	}
	if f.Order == "" {
		f.Order = expense.OrderDesc
	}
	if f.Limit == 0 {
		f.Limit = expense.DefaultLimit
	}
	return f
}

func testSortAndPagination(t *testing.T, r Repos) {
	ids := seed(t, r)
	tests := []struct {
		name string
		f    expense.Filter
		want []int
	}{
		{"spent_at desc, ties by id desc", expense.Filter{Sort: expense.SortSpentAt, Order: expense.OrderDesc}, []int{0, 2, 1, 3}},
		{"spent_at asc, ties by id asc", expense.Filter{Sort: expense.SortSpentAt, Order: expense.OrderAsc}, []int{3, 1, 2, 0}},
		{"amount desc", expense.Filter{Sort: expense.SortAmount, Order: expense.OrderDesc}, []int{2, 3, 1, 0}},
		{"amount asc", expense.Filter{Sort: expense.SortAmount, Order: expense.OrderAsc}, []int{0, 1, 3, 2}},
		{"created_at desc", expense.Filter{Sort: expense.SortCreatedAt, Order: expense.OrderDesc}, []int{3, 2, 1, 0}},
		{"created_at asc", expense.Filter{Sort: expense.SortCreatedAt, Order: expense.OrderAsc}, []int{0, 1, 2, 3}},
		{"first page", expense.Filter{Limit: 2}, []int{0, 2}},
		{"second page", expense.Filter{Limit: 2, Offset: 2}, []int{1, 3}},
		{"past the end", expense.Filter{Limit: 2, Offset: 4}, []int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listIDs(t, r, tt.f)
			if want := pick(ids, tt.want...); !slices.Equal(got, want) {
				t.Errorf("ids = %v, want %v", got, want)
			}
		})
	}
}

func testSummary(t *testing.T, r Repos) {
	ctx := context.Background()
	seed(t, r)

	sum, err := r.Expenses.Summary(ctx, withDefaults(expense.Filter{}))
	if err != nil {
		t.Fatal(err)
	}
	if sum.Total != 623050 || sum.Count != 4 {
		t.Errorf("Summary total = %+v", sum)
	}
	// food 3830.50 first; transport and health tie at 1200 and keep display order.
	want := []expense.CategoryTotal{
		{Category: expense.CategoryFood, Total: 383050, Count: 2},
		{Category: expense.CategoryTransport, Total: 120000, Count: 1},
		{Category: expense.CategoryHealth, Total: 120000, Count: 1},
	}
	if !slices.Equal(sum.ByCategory, want) {
		t.Errorf("ByCategory = %+v, want %+v", sum.ByCategory, want)
	}

	sum, err = r.Expenses.Summary(ctx, withDefaults(expense.Filter{UserID: ptr(Alisher), Query: "кофе"}))
	if err != nil || sum.Total != 35000 || sum.Count != 1 {
		t.Errorf("filtered Summary = %+v, %v", sum, err)
	}

	empty, err := r.Expenses.Summary(ctx, withDefaults(expense.Filter{UserID: ptr(Stranger)}))
	if err != nil || empty.Total != 0 || empty.Count != 0 || empty.ByCategory == nil || len(empty.ByCategory) != 0 {
		t.Errorf("empty Summary = %+v, %v (ByCategory must be empty, not nil)", empty, err)
	}
}
