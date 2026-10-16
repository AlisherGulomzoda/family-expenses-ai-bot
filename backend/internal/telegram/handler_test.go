package telegram

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

type fakeSender struct {
	mu     sync.Mutex
	texts  []string
	typing int
}

func (s *fakeSender) SendText(_ context.Context, _ int64, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.texts = append(s.texts, text)
	return nil
}

func (s *fakeSender) SendTyping(context.Context, int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.typing++
	return nil
}

type fakeAgent struct {
	reply string
	err   error
	delay time.Duration
	got   []user.User
	texts []string
}

func (a *fakeAgent) Handle(_ context.Context, u user.User, text string) (string, error) {
	time.Sleep(a.delay)
	a.got = append(a.got, u)
	a.texts = append(a.texts, text)
	return a.reply, a.err
}

type fakeUsers struct {
	err error
	got []user.User
}

func (f *fakeUsers) Upsert(_ context.Context, u user.User) (user.User, error) {
	f.got = append(f.got, u)
	if u.Name == "" {
		u.Name = "fallback"
	}
	return u, f.err
}

const family int64 = 111

func setup(allowed ...int64) (*Handler, *fakeAgent, *fakeUsers, *fakeSender) {
	a := &fakeAgent{reply: "Записал: кофе — 350 ₽."}
	u := &fakeUsers{}
	h := NewHandler(a, u, allowed, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return h, a, u, &fakeSender{}
}

func msg(text string) Incoming {
	return Incoming{ChatID: 1, Private: true, FromID: family, FirstName: "Алишер", LastName: "Г", Username: "alisher", Text: text}
}

func TestMessageGoesToAgentAsTheSender(t *testing.T) {
	h, a, users, out := setup(family)
	h.Handle(t.Context(), out, msg("  кофе 350  "))

	if len(out.texts) != 1 || out.texts[0] != "Записал: кофе — 350 ₽." {
		t.Fatalf("sent = %q", out.texts)
	}
	if len(a.got) != 1 || a.got[0].TelegramID != family || a.texts[0] != "кофе 350" {
		t.Errorf("agent got %+v %q", a.got, a.texts)
	}
	u := users.got[0]
	if u.TelegramID != family || u.Name != "Алишер Г" || u.Username == nil || *u.Username != "alisher" {
		t.Errorf("registered %+v", u)
	}
	if out.typing == 0 {
		t.Error("no typing indicator")
	}
}

func TestWhitelist(t *testing.T) {
	h, a, users, out := setup(family)
	stranger := msg("кофе 350")
	stranger.FromID = 999
	h.Handle(t.Context(), out, stranger)

	if len(out.texts) != 1 || out.texts[0] != ReplyNotAllowed {
		t.Errorf("sent = %q", out.texts)
	}
	if len(a.got) != 0 || len(users.got) != 0 {
		t.Error("a stranger reached the agent or was registered")
	}

	// An empty whitelist lets everyone in.
	h, a, _, out = setup()
	h.Handle(t.Context(), out, stranger)
	if len(a.got) != 1 {
		t.Errorf("empty whitelist: agent calls = %d", len(a.got))
	}
}

func TestRepliesWithoutTheAgent(t *testing.T) {
	tests := []struct {
		name string
		in   func(Incoming) Incoming
		want string
	}{
		{"start", func(in Incoming) Incoming { in.Text = "/start"; return in }, ReplyStart},
		{"start with bot name", func(in Incoming) Incoming { in.Text = "/start@family_bot"; return in }, ReplyStart},
		{"help", func(in Incoming) Incoming { in.Text = "/help"; return in }, ReplyStart},
		{"sticker or photo", func(in Incoming) Incoming { in.Text = ""; return in }, ReplyOnlyText},
		{"group chat", func(in Incoming) Incoming { in.Private = false; return in }, ReplyOnlyPrivate},
		{"too long", func(in Incoming) Incoming { in.Text = strings.Repeat("кофе ", 300); return in }, ReplyTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, a, _, out := setup(family)
			h.Handle(t.Context(), out, tt.in(msg("кофе 350")))
			if len(out.texts) != 1 || out.texts[0] != tt.want {
				t.Errorf("sent = %q, want %q", out.texts, tt.want)
			}
			if len(a.got) != 0 {
				t.Error("the agent was called")
			}
		})
	}
}

func TestIgnoresBotsAndAnonymous(t *testing.T) {
	h, a, _, out := setup()
	fromBot := msg("кофе 350")
	fromBot.FromBot = true
	h.Handle(t.Context(), out, fromBot)
	anonymous := msg("кофе 350")
	anonymous.FromID = 0
	h.Handle(t.Context(), out, anonymous)
	if len(out.texts) != 0 || len(a.got) != 0 {
		t.Errorf("sent = %q, agent calls = %d", out.texts, len(a.got))
	}
}

func TestAgentErrorStillAnswers(t *testing.T) {
	h, a, _, out := setup(family)
	a.reply, a.err = "Сервис недоступен, попробуй позже.", errors.New("llm: 402")
	h.Handle(t.Context(), out, msg("кофе 350"))
	if len(out.texts) != 1 || out.texts[0] != a.reply {
		t.Errorf("sent = %q", out.texts)
	}
}

func TestUserRegistrationFailure(t *testing.T) {
	h, a, users, out := setup(family)
	users.err = errors.New("db down")
	h.Handle(t.Context(), out, msg("кофе 350"))
	if len(out.texts) != 1 || out.texts[0] != ReplyInternalFail || len(a.got) != 0 {
		t.Errorf("sent = %q, agent calls = %d", out.texts, len(a.got))
	}
}

func TestTypingIsRefreshedForSlowAnswers(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the typing refresh")
	}
	h, a, _, out := setup(family)
	a.delay = typingEvery + 500*time.Millisecond
	h.Handle(t.Context(), out, msg("покажи траты за месяц"))
	if out.typing < 2 {
		t.Errorf("typing sent %d times, want a refresh", out.typing)
	}
}

func TestLongRepliesAreSplit(t *testing.T) {
	h, a, _, out := setup(family)
	line := strings.Repeat("ж", 99) + "\n" // 100 runes, 199 bytes
	a.reply = strings.Repeat(line, 100)    // 10 000 runes
	h.Handle(t.Context(), out, msg("покажи всё"))

	if len(out.texts) != 3 {
		t.Fatalf("parts = %d, want 3", len(out.texts))
	}
	total := 0
	for _, p := range out.texts {
		n := utf8.RuneCountInString(p)
		if n > maxMessageRunes || !utf8.ValidString(p) {
			t.Errorf("part of %d runes (valid UTF-8: %v)", n, utf8.ValidString(p))
		}
		if strings.HasSuffix(p, "\n") {
			t.Error("part ends with a dangling newline")
		}
		total += strings.Count(p, "ж")
	}
	if total != 9900 {
		t.Errorf("lost characters: %d", total)
	}
}

func TestSplit(t *testing.T) {
	if got := split("", 10); got != nil {
		t.Errorf("empty = %q", got)
	}
	if got := split("короткий", 10); len(got) != 1 || got[0] != "короткий" {
		t.Errorf("short = %q", got)
	}
	// No line breaks: hard cut by runes, never inside a character.
	got := split(strings.Repeat("я", 25), 10)
	if len(got) != 3 || got[0] != strings.Repeat("я", 10) || got[2] != strings.Repeat("я", 5) {
		t.Errorf("hard cut = %q", got)
	}
}

func TestCommand(t *testing.T) {
	for in, want := range map[string]string{
		"/start": "/start", "/START": "/start", "/start@bot": "/start", "/help me": "/help",
		"кофе 350": "", "": "",
	} {
		if got := command(in); got != want {
			t.Errorf("command(%q) = %q, want %q", in, got, want)
		}
	}
}
