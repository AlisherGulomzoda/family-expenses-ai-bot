// Package user manages family members: people allowed to use the bot.
package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// User is a family member identified by their Telegram ID.
type User struct {
	TelegramID int64
	Name       string
	Username   *string // Telegram @username without "@"
}

// Repository stores users.
type Repository interface {
	// Upsert creates the user or refreshes their name and username.
	Upsert(ctx context.Context, u User) (User, error)
	// List returns all users ordered by name.
	List(ctx context.Context) ([]User, error)
}

// ErrInvalidUser is returned for a user without a Telegram ID.
var ErrInvalidUser = errors.New("invalid user")

// Service is the family member use cases.
type Service struct {
	repo Repository
}

// NewService creates a Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Upsert registers a family member when they write to the bot, keeping
// their name in sync with the Telegram profile.
func (s *Service) Upsert(ctx context.Context, u User) (User, error) {
	if u.TelegramID <= 0 {
		return User{}, fmt.Errorf("%w: telegram id %d", ErrInvalidUser, u.TelegramID)
	}
	if u.Username != nil {
		name := strings.TrimPrefix(strings.TrimSpace(*u.Username), "@")
		u.Username = &name
		if name == "" {
			u.Username = nil
		}
	}
	u.Name = strings.TrimSpace(u.Name)
	if u.Name == "" {
		u.Name = fallbackName(u)
	}
	return s.repo.Upsert(ctx, u)
}

// List returns all family members ordered by name.
func (s *Service) List(ctx context.Context) ([]User, error) {
	return s.repo.List(ctx)
}

// fallbackName is used when the Telegram profile has no name.
func fallbackName(u User) string {
	if u.Username != nil {
		return "@" + *u.Username
	}
	return "Пользователь " + strconv.FormatInt(u.TelegramID, 10)
}
