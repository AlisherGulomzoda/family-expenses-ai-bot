// Package agent turns a user's free-form message into expense operations:
// it runs the LLM tool-calling loop over the expense tools and keeps the
// conversation history.
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent/tools"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

// Replies used when the model cannot produce one.
const (
	ReplyLLMUnavailable = "Не получилось обработать сообщение: сервис распознавания недоступен. Попробуй ещё раз через минуту."
	ReplyTooManySteps   = "Не получилось выполнить запрос до конца. Попробуй сформулировать проще, например: «кофе 350»."
	ReplyInternalError  = "Что-то пошло не так. Попробуй ещё раз через минуту."
	replyDone           = "Готово."
	replyNotUnderstood  = "Не понял. Напиши, например: «кофе 350» или «сколько я потратил за неделю»."
)

// Expenses gives the expense operations of one user.
type Expenses interface {
	ForUser(userID int64) expense.UserScope
}

// Config tunes the agent.
type Config struct {
	// HistoryLimit is how many stored messages the model sees.
	HistoryLimit int
	// MaxSteps bounds the LLM round trips per user message.
	MaxSteps int
}

// Agent handles user messages.
type Agent struct {
	llm      llm.Client
	tools    *tools.Registry
	expenses Expenses
	history  History
	cfg      Config
	log      *slog.Logger

	mu    sync.Mutex
	locks map[int64]*sync.Mutex
}

// New creates an Agent.
func New(client llm.Client, reg *tools.Registry, expenses Expenses, history History, cfg Config, log *slog.Logger) *Agent {
	if cfg.MaxSteps < 1 {
		cfg.MaxSteps = 1
	}
	return &Agent{
		llm: client, tools: reg, expenses: expenses, history: history, cfg: cfg, log: log,
		locks: map[int64]*sync.Mutex{},
	}
}

// Handle answers one message of u. The returned reply is always suitable
// for the user, also when err is not nil (err is for logging).
//
// Messages of one user are handled one at a time, so their turns never
// interleave in the history.
func (a *Agent) Handle(ctx context.Context, u user.User, text string) (string, error) {
	unlock := a.lock(u.TelegramID)
	defer unlock()

	start := time.Now()
	scope := a.expenses.ForUser(u.TelegramID)
	t := &turn{agent: a, scope: scope, userID: u.TelegramID}
	reply, err := t.run(ctx, u, text)

	a.log.InfoContext(ctx, "agent turn",
		"user_id", u.TelegramID,
		"steps", t.steps,
		"tools", t.toolNames,
		"tool_errors", t.toolErrors,
		"prompt_tokens", t.usage.PromptTokens,
		"completion_tokens", t.usage.CompletionTokens,
		"duration_ms", time.Since(start).Milliseconds(),
		"error", err,
	)
	return reply, err
}

func (a *Agent) lock(userID int64) func() {
	a.mu.Lock()
	m, ok := a.locks[userID]
	if !ok {
		m = &sync.Mutex{}
		a.locks[userID] = m
	}
	a.mu.Unlock()
	m.Lock()
	return m.Unlock
}

// turn is the state of handling one user message.
type turn struct {
	agent      *Agent
	scope      expense.UserScope
	userID     int64
	steps      int
	toolNames  []string
	toolErrors int
	usage      llm.Usage
}

func (t *turn) run(ctx context.Context, u user.User, text string) (string, error) {
	a := t.agent
	prompt, err := systemPrompt(u.Name, t.scope.Location().String(), t.scope.Today())
	if err != nil {
		return ReplyInternalError, err
	}
	past, err := a.history.Last(ctx, t.userID, a.cfg.HistoryLimit)
	if err != nil {
		return ReplyInternalError, fmt.Errorf("load history: %w", err)
	}

	userMsg := llm.Message{Role: llm.RoleUser, Content: text}
	msgs := make([]llm.Message, 0, len(past)+8)
	msgs = append(msgs, llm.Message{Role: llm.RoleSystem, Content: prompt})
	msgs = append(msgs, trimToTurn(past)...)
	msgs = append(msgs, userMsg)
	// newMsgs is what this turn adds to the history.
	newMsgs := []llm.Message{userMsg}
	defs := a.tools.Defs()

	for t.steps < a.cfg.MaxSteps {
		t.steps++
		resp, err := a.llm.Chat(ctx, llm.Request{Messages: msgs, Tools: defs})
		if err != nil {
			// Nothing is stored: the user will simply resend the message.
			return ReplyLLMUnavailable, fmt.Errorf("llm: %w", err)
		}
		t.usage.PromptTokens += resp.Usage.PromptTokens
		t.usage.CompletionTokens += resp.Usage.CompletionTokens
		msgs = append(msgs, resp.Message)
		newMsgs = append(newMsgs, resp.Message)

		if len(resp.Message.ToolCalls) == 0 {
			reply := resp.Message.Content
			if reply == "" {
				reply = replyNotUnderstood
				if len(t.toolNames) > 0 {
					reply = replyDone
				}
			}
			return reply, t.save(ctx, newMsgs)
		}

		for _, call := range resp.Message.ToolCalls {
			res := a.tools.Call(ctx, t.scope, call.Name, call.Arguments)
			t.toolNames = append(t.toolNames, call.Name)
			if res.Err != nil {
				t.toolErrors++
				level := slog.LevelWarn
				if res.Internal {
					level = slog.LevelError
				}
				a.log.Log(ctx, level, "tool call failed",
					"user_id", t.userID, "tool", call.Name, "arguments", call.Arguments, "error", res.Err)
			}
			toolMsg := llm.Message{Role: llm.RoleTool, ToolCallID: call.ID, Content: res.Content}
			msgs = append(msgs, toolMsg)
			newMsgs = append(newMsgs, toolMsg)
		}
	}

	// Out of steps: close the turn with an answer so the stored history
	// stays a valid conversation.
	newMsgs = append(newMsgs, llm.Message{Role: llm.RoleAssistant, Content: ReplyTooManySteps})
	if err := t.save(ctx, newMsgs); err != nil {
		return ReplyTooManySteps, err
	}
	return ReplyTooManySteps, fmt.Errorf("agent: no answer after %d steps", t.steps)
}

func (t *turn) save(ctx context.Context, msgs []llm.Message) error {
	if err := t.agent.history.Append(ctx, t.userID, msgs); err != nil {
		return fmt.Errorf("save history: %w", err)
	}
	return nil
}
