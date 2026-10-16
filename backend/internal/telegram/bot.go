package telegram

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Run receives updates by long polling (no public URL needed) and passes
// messages to h until ctx is done.
func Run(ctx context.Context, token string, h *Handler, log *slog.Logger) error {
	b, err := bot.New(token,
		bot.WithAllowedUpdates(bot.AllowedUpdates{"message"}),
		bot.WithErrorsHandler(func(err error) {
			log.WarnContext(ctx, "telegram polling error", "error", err)
		}),
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message == nil {
				return
			}
			h.Handle(ctx, sender{b}, incoming(update.Message))
		}),
	)
	if err != nil {
		// bot.New calls getMe: a wrong token fails here, at startup.
		return fmt.Errorf("connect to telegram: %w", err)
	}
	log.InfoContext(ctx, "telegram bot started (long polling)")
	b.Start(ctx) // blocks until ctx is done
	return nil
}

func incoming(m *models.Message) Incoming {
	in := Incoming{
		ChatID:  m.Chat.ID,
		Private: m.Chat.Type == models.ChatTypePrivate,
		Text:    m.Text,
	}
	if m.From != nil {
		in.FromID = m.From.ID
		in.FromBot = m.From.IsBot
		in.FirstName = m.From.FirstName
		in.LastName = m.From.LastName
		in.Username = m.From.Username
	}
	return in
}

// sender sends plain text: no parse mode, so user and model text with
// "*", "_" or "<" can never break the formatting or the request.
type sender struct{ b *bot.Bot }

func (s sender) SendText(ctx context.Context, chatID int64, text string) error {
	_, err := s.b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text})
	return err
}

func (s sender) SendTyping(ctx context.Context, chatID int64) error {
	_, err := s.b.SendChatAction(ctx, &bot.SendChatActionParams{ChatID: chatID, Action: models.ChatActionTyping})
	return err
}
