package memstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense/expensetest"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/memstore"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

var now = clock.Fixed(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC))

func TestRepositoryContract(t *testing.T) {
	expensetest.RunRepositoryContract(t, func(*testing.T) expensetest.Repos {
		s := memstore.New(now)
		return expensetest.Repos{Expenses: s.Expenses(), Users: s.Users()}
	})
}

func TestReturnedValuesAreCopies(t *testing.T) {
	ctx := context.Background()
	s := memstore.New(now)
	if _, err := s.Users().Upsert(ctx, user.User{TelegramID: 1, Name: "Алишер"}); err != nil {
		t.Fatal(err)
	}
	comment := "Кофе"
	e, err := s.Expenses().Create(ctx, expense.NewExpense{
		UserID: 1, Amount: 100, Category: expense.CategoryFood, Comment: &comment, Source: expense.SourceBot,
	})
	if err != nil {
		t.Fatal(err)
	}
	comment = "changed by caller input"
	*e.Comment = "changed by caller output"

	again, err := s.Expenses().Get(ctx, e.ID, nil)
	if err != nil || *again.Comment != "Кофе" {
		t.Errorf("stored comment = %q, %v; store data leaked through a pointer", *again.Comment, err)
	}
}
