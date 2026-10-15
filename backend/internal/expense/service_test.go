package expense_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/memstore"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

const (
	alisher int64 = 100000001
	madina  int64 = 100000002
)

// 2026-10-15 22:30 UTC is already 2026-10-16 01:30 in Moscow.
var now = time.Date(2026, 10, 15, 22, 30, 0, 0, time.UTC)

func newService(t *testing.T) *expense.Service {
	t.Helper()
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	store := memstore.New(clock.Fixed(now))
	for _, u := range []user.User{{TelegramID: alisher, Name: "Алишер"}, {TelegramID: madina, Name: "Мадина"}} {
		if _, err := store.Users().Upsert(context.Background(), u); err != nil {
			t.Fatal(err)
		}
	}
	return expense.NewService(store.Expenses(), clock.Fixed(now), moscow)
}

func ptr[T any](v T) *T { return &v }

func TestTodayUsesServiceTimeZone(t *testing.T) {
	svc := newService(t)
	e, err := svc.ForUser(alisher).Create(context.Background(), expense.Draft{Amount: 35000, Category: expense.CategoryFood})
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC); !e.SpentAt.Equal(want) {
		t.Errorf("SpentAt = %v, want %v (Moscow date, not UTC)", e.SpentAt, want)
	}
}

func TestSourceIsSetByService(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)

	web, err := svc.Create(ctx, expense.CreateInput{UserID: madina, Draft: expense.Draft{Amount: 89000, Category: expense.CategoryHealth}})
	if err != nil {
		t.Fatal(err)
	}
	if web.Source != expense.SourceWeb || web.UserName != "Мадина" {
		t.Errorf("web expense = %+v", web)
	}

	bot, err := svc.ForUser(alisher).Create(ctx, expense.Draft{Amount: 35000, Category: expense.CategoryFood})
	if err != nil {
		t.Fatal(err)
	}
	if bot.Source != expense.SourceBot || bot.UserID != alisher {
		t.Errorf("bot expense = %+v", bot)
	}

	// Editing through the web panel does not turn a bot expense into a web one.
	updated, err := svc.Update(ctx, bot.ID, expense.UpdateInput{Patch: expense.Patch{Amount: ptr(expense.Money(45000))}})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Source != expense.SourceBot || updated.Amount != 45000 {
		t.Errorf("updated = %+v", updated)
	}
}

func TestUserScopeIsolation(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	asAlisher, asMadina := svc.ForUser(alisher), svc.ForUser(madina)

	taxi, err := asMadina.Create(ctx, expense.Draft{Amount: 120000, Category: expense.CategoryTransport, Comment: ptr("такси")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := asAlisher.Create(ctx, expense.Draft{Amount: 35000, Category: expense.CategoryFood, Comment: ptr("кофе")}); err != nil {
		t.Fatal(err)
	}

	if _, err := asAlisher.Get(ctx, taxi.ID); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Get foreign: err = %v, want ErrNotFound", err)
	}
	if _, err := asAlisher.Update(ctx, taxi.ID, expense.Patch{Amount: ptr(expense.Money(1))}); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Update foreign: err = %v, want ErrNotFound", err)
	}
	if _, err := asAlisher.Update(ctx, taxi.ID, expense.Patch{}); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("empty Update foreign: err = %v, want ErrNotFound", err)
	}
	if _, err := asAlisher.Delete(ctx, taxi.ID); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Delete foreign: err = %v, want ErrNotFound", err)
	}

	// Madina's expense is untouched.
	got, err := asMadina.Get(ctx, taxi.ID)
	if err != nil || got.Amount != 120000 {
		t.Fatalf("Madina's taxi = %+v, %v", got, err)
	}

	// Asking for someone else's expenses in the filter is ignored.
	page, err := asAlisher.List(ctx, expense.Filter{UserID: ptr(madina)})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].UserID != alisher {
		t.Errorf("Alisher sees %+v", page)
	}
	sum, err := asAlisher.Summary(ctx, expense.Filter{UserID: ptr(madina)})
	if err != nil {
		t.Fatal(err)
	}
	if sum.Total != 35000 || sum.Count != 1 {
		t.Errorf("Alisher summary = %+v", sum)
	}

	// The owner can delete and gets back what was deleted.
	deleted, err := asMadina.Delete(ctx, taxi.ID)
	if err != nil || deleted.ID != taxi.ID || deleted.Comment == nil || *deleted.Comment != "такси" {
		t.Fatalf("Delete own = %+v, %v", deleted, err)
	}
}

func TestAdminSeesEveryone(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)
	for _, id := range []int64{alisher, madina} {
		if _, err := svc.ForUser(id).Create(ctx, expense.Draft{Amount: 10000, Category: expense.CategoryOther}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := svc.List(ctx, expense.Filter{})
	if err != nil || page.Total != 2 {
		t.Fatalf("List = %+v, %v", page, err)
	}
	page, err = svc.List(ctx, expense.Filter{UserID: ptr(madina)})
	if err != nil || page.Total != 1 || page.Items[0].UserID != madina {
		t.Fatalf("List(madina) = %+v, %v", page, err)
	}

	// Move Madina's expense to Alisher.
	moved, err := svc.Update(ctx, page.Items[0].ID, expense.UpdateInput{UserID: ptr(alisher)})
	if err != nil || moved.UserID != alisher || moved.UserName != "Алишер" {
		t.Fatalf("move = %+v, %v", moved, err)
	}

	if err := svc.Delete(ctx, moved.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, moved.ID); !errors.Is(err, expense.ErrNotFound) {
		t.Errorf("Get deleted: %v", err)
	}
}

func TestCreateValidation(t *testing.T) {
	ctx := context.Background()
	svc := newService(t)

	_, err := svc.Create(ctx, expense.CreateInput{Draft: expense.Draft{Amount: 0, Category: "fun"}})
	var ve *expense.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	for _, f := range []string{"user_id", "amount", "category"} {
		if _, ok := ve.Fields[f]; !ok {
			t.Errorf("missing %q in %v", f, ve.Fields)
		}
	}

	// Unknown family member is reported by the repository as a field error.
	_, err = svc.Create(ctx, expense.CreateInput{UserID: 42, Draft: expense.Draft{Amount: 100, Category: expense.CategoryFood}})
	if !errors.As(err, &ve) || ve.Fields["user_id"] == "" {
		t.Fatalf("unknown user: err = %v", err)
	}

	if _, err := svc.List(ctx, expense.Filter{Limit: 1000}); !errors.As(err, &ve) {
		t.Errorf("bad filter: err = %v", err)
	}
}
