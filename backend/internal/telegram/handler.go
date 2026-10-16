// Package telegram connects the agent to Telegram: who may talk to the bot,
// what it answers without the LLM, and how replies are delivered.
//
// Handler holds the logic and knows nothing about the Telegram library;
// Run (bot.go) adapts it to long polling.
package telegram

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

// Bot replies. Kept short: they are read on a phone.
const (
	ReplyStart = "Привет! Я помогаю вести учёт семейных расходов. Пиши обычным текстом:\n" +
		"• «кофе 350» или «такси 1200 вчера» — запишу трату;\n" +
		"• «сколько я потратил на еду в этом месяце» или «покажи траты за неделю»;\n" +
		"• «последнее было не 350, а 450» — исправлю;\n" +
		"• «удали такси» — удалю.\n" +
		"Я вижу только твои траты."
	ReplyNotAllowed   = "Этот бот доступен только членам семьи. Попроси главу семьи добавить тебя."
	ReplyOnlyText     = "Я понимаю только текст. Напиши, например: «кофе 350»."
	ReplyOnlyPrivate  = "Я работаю только в личных сообщениях."
	ReplyTooLong      = "Сообщение слишком длинное. Напиши короче, например: «кофе 350»."
	ReplyInternalFail = "Что-то пошло не так. Попробуй ещё раз через минуту."
)

const (
	// maxInputRunes protects the LLM budget from pasted essays.
	maxInputRunes = 1000
	// maxMessageRunes is Telegram's limit for one text message.
	maxMessageRunes = 4096
	// typingEvery refreshes "typing…": Telegram shows it for about 5 seconds.
	typingEvery = 4 * time.Second
)

// Incoming is a Telegram message reduced to what the bot needs.
type Incoming struct {
	ChatID    int64
	Private   bool
	FromID    int64
	FromBot   bool
	FirstName string
	LastName  string
	Username  string
	// Text is empty for stickers, photos, voice messages and so on.
	Text string
}

// Sender delivers messages to Telegram.
type Sender interface {
	SendText(ctx context.Context, chatID int64, text string) error
	SendTyping(ctx context.Context, chatID int64) error
}

// Agent answers a user's message.
type Agent interface {
	Handle(ctx context.Context, u user.User, text string) (string, error)
}

// Users registers family members.
type Users interface {
	Upsert(ctx context.Context, u user.User) (user.User, error)
}

// Handler processes incoming messages.
type Handler struct {
	agent   Agent
	users   Users
	allowed []int64 // empty: everyone
	log     *slog.Logger
}

// NewHandler creates a Handler. allowed is the family whitelist of
// Telegram user IDs; empty allows everyone.
func NewHandler(agent Agent, users Users, allowed []int64, log *slog.Logger) *Handler {
	return &Handler{agent: agent, users: users, allowed: allowed, log: log}
}

// Handle answers one message. The sender's identity comes from Telegram
// (in.FromID) and is the only identity the agent ever works with.
func (h *Handler) Handle(ctx context.Context, out Sender, in Incoming) {
	if in.FromBot || in.FromID == 0 {
		return
	}
	reply := func(text string) {
		for _, part := range split(text, maxMessageRunes) {
			if err := out.SendText(ctx, in.ChatID, part); err != nil {
				h.log.ErrorContext(ctx, "send telegram message", "chat_id", in.ChatID, "error", err)
				return
			}
		}
	}

	switch {
	case !in.Private:
		reply(ReplyOnlyPrivate)
		return
	case !h.isAllowed(in.FromID):
		// The ID in the log is what the family head adds to ALLOWED_TELEGRAM_IDS.
		h.log.WarnContext(ctx, "message from a user outside the family whitelist",
			"user_id", in.FromID, "username", in.Username)
		reply(ReplyNotAllowed)
		return
	}

	text := strings.TrimSpace(in.Text)
	switch command(text) {
	case "/start", "/help":
		reply(ReplyStart)
		return
	}
	switch {
	case text == "":
		reply(ReplyOnlyText)
		return
	case utf8.RuneCountInString(text) > maxInputRunes:
		reply(ReplyTooLong)
		return
	}

	u, err := h.users.Upsert(ctx, user.User{
		TelegramID: in.FromID,
		Name:       strings.TrimSpace(in.FirstName + " " + in.LastName),
		Username:   optional(in.Username),
	})
	if err != nil {
		h.log.ErrorContext(ctx, "register user", "user_id", in.FromID, "error", err)
		reply(ReplyInternalFail)
		return
	}

	stopTyping := h.keepTyping(ctx, out, in.ChatID)
	answer, err := h.agent.Handle(ctx, u, text)
	stopTyping()
	if err != nil {
		h.log.ErrorContext(ctx, "agent failed", "user_id", in.FromID, "error", err)
	}
	reply(answer)
}

func (h *Handler) isAllowed(id int64) bool {
	return len(h.allowed) == 0 || slices.Contains(h.allowed, id)
}

// keepTyping shows "typing…" until the returned stop function is called.
func (h *Handler) keepTyping(ctx context.Context, out Sender, chatID int64) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(typingEvery)
		defer ticker.Stop()
		for {
			if err := out.SendTyping(ctx, chatID); err != nil && ctx.Err() == nil {
				h.log.DebugContext(ctx, "send typing", "error", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

// command returns the bot command of text ("/start@my_bot args" -> "/start").
func command(text string) string {
	if !strings.HasPrefix(text, "/") {
		return ""
	}
	cmd, _, _ := strings.Cut(text, " ")
	cmd, _, _ = strings.Cut(cmd, "@")
	return strings.ToLower(cmd)
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// split cuts text into parts of at most limit runes, preferring line breaks.
// Surrounding whitespace (models often end with a newline) is dropped.
func split(text string, limit int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	var parts []string
	runes := []rune(text)
	for len(runes) > limit {
		cut := limit
		if i := lastIndex(runes[:limit], '\n'); i > limit/2 {
			cut = i + 1
		}
		parts = append(parts, strings.TrimRight(string(runes[:cut]), "\n"))
		runes = runes[cut:]
	}
	return append(parts, string(runes))
}

func lastIndex(runes []rune, r rune) int {
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == r {
			return i
		}
	}
	return -1
}
