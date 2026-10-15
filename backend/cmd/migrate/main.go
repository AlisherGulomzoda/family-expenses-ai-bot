// Command migrate applies database migrations and exits.
package main

import (
	"fmt"
	"os"
	_ "time/tzdata" // the runtime image has no system time zone database

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/config"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load[config.Migrate]()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, "migrate")

	// Migrations are added in BE-02.
	log.Info("no migrations to apply")
	return nil
}
