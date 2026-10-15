package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/memstore"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

func ptr[T any](v T) *T { return &v }

func TestUpsert(t *testing.T) {
	ctx := context.Background()
	svc := user.NewService(memstore.New(clock.Fixed(time.Now())).Users())

	tests := []struct {
		name         string
		in           user.User
		wantName     string
		wantUsername *string
	}{
		{"trims name and @", user.User{TelegramID: 1, Name: "  Алишер ", Username: ptr("@alisher")}, "Алишер", ptr("alisher")},
		{"no name falls back to username", user.User{TelegramID: 2, Username: ptr("madina")}, "@madina", ptr("madina")},
		{"nothing falls back to id", user.User{TelegramID: 3, Username: ptr(" ")}, "Пользователь 3", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.Upsert(ctx, tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if (got.Username == nil) != (tt.wantUsername == nil) || (got.Username != nil && *got.Username != *tt.wantUsername) {
				t.Errorf("Username = %v, want %v", got.Username, tt.wantUsername)
			}
		})
	}

	if _, err := svc.Upsert(ctx, user.User{Name: "no id"}); !errors.Is(err, user.ErrInvalidUser) {
		t.Errorf("Upsert without id: %v", err)
	}

	users, err := svc.List(ctx)
	if err != nil || len(users) != 3 || users[0].Name != "@madina" {
		t.Errorf("List = %+v, %v", users, err)
	}
}
