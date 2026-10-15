// Command bot runs the Telegram bot with the LLM expense agent.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata" // the runtime image has no system time zone database

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/config"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/logger"
)

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

	if len(cfg.AllowedIDs) == 0 {
		log.Warn("ALLOWED_TELEGRAM_IDS is empty: the bot will answer anyone")
	}
	log.Info("bot started", "llm_base_url", cfg.LLM.BaseURL, "llm_model", cfg.LLM.Model)
	<-ctx.Done()
	log.Info("bot stopped")
	return nil
}
