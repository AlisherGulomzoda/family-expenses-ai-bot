// Command bot runs the Telegram bot with the LLM expense agent.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata" // the runtime image has no system time zone database

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent/tools"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/config"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/llm"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/logger"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/postgres"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/telegram"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

// llmMaxRetries is the number of retries on rate limits and server errors.
const llmMaxRetries = 2

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "bot:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load[config.Bot]()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, "bot")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(ctx, cfg.DB.URL)
	if err != nil {
		return err
	}
	defer pool.Close()

	expenses := expense.NewService(postgres.NewExpenseRepo(pool), clock.System{}, cfg.Timezone.Location)
	users := user.NewService(postgres.NewUserRepo(pool))
	model := llm.NewOpenAI(llm.OpenAIConfig{
		BaseURL:     cfg.LLM.BaseURL,
		APIKey:      cfg.LLM.APIKey,
		Model:       cfg.LLM.Model,
		Temperature: cfg.LLM.Temperature,
		Timeout:     cfg.LLM.Timeout,
		MaxRetries:  llmMaxRetries,
		Log:         log,
	})
	expenseAgent := agent.New(model, tools.Default(), expenses, postgres.NewChatHistoryRepo(pool),
		agent.Config{HistoryLimit: cfg.HistoryLimit, MaxSteps: cfg.MaxSteps}, log)

	if len(cfg.AllowedIDs) == 0 {
		log.Warn("ALLOWED_TELEGRAM_IDS is empty: the bot will answer anyone")
	}
	log.Info("bot starting", "llm_base_url", cfg.LLM.BaseURL, "llm_model", cfg.LLM.Model,
		"family_members", len(cfg.AllowedIDs), "timezone", cfg.Timezone.String())

	handler := telegram.NewHandler(expenseAgent, users, cfg.AllowedIDs, log)
	if err := telegram.Run(ctx, cfg.Telegram.Token, handler, log); err != nil {
		return err
	}
	log.Info("bot stopped")
	return nil
}
