// Command api serves the REST API for the family head's web panel.
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
		fmt.Fprintln(os.Stderr, "api:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load[config.API]()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, "api")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("api started", "addr", cfg.HTTPAddr, "timezone", cfg.Timezone.String())
	<-ctx.Done()
	log.Info("api stopped")
	return nil
}
