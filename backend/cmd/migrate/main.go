// Command migrate manages the database schema and exits.
//
// Usage:
//
//	migrate [up|down|reset|status]
//
//	up      apply all pending migrations (default)
//	down    roll back the last applied migration
//	reset   roll back all migrations
//	status  print the state of every migration
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // the runtime image has no system time zone database

	"github.com/pressly/goose/v3"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/config"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/logger"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/postgres"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run() error {
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	cfg, err := config.Load[config.Migrate]()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, "migrate")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	db, err := postgres.OpenSQL(cfg.DB.URL)
	if err != nil {
		return err
	}
	defer db.Close()

	migrator, err := postgres.NewMigrator(db)
	if err != nil {
		return err
	}

	switch cmd {
	case "up":
		results, err := migrator.Up(ctx)
		logResults(log, results)
		if err != nil {
			return fmt.Errorf("up: %w", err)
		}
	case "down":
		result, err := migrator.Down(ctx)
		if result != nil {
			logResults(log, []*goose.MigrationResult{result})
		}
		if err != nil {
			return fmt.Errorf("down: %w", err)
		}
	case "reset":
		results, err := migrator.DownTo(ctx, 0)
		logResults(log, results)
		if err != nil {
			return fmt.Errorf("reset: %w", err)
		}
	case "status":
		statuses, err := migrator.Status(ctx)
		if err != nil {
			return fmt.Errorf("status: %w", err)
		}
		for _, s := range statuses {
			log.Info("migration", "version", s.Source.Version, "file", s.Source.Path, "state", s.State)
		}
		return nil
	default:
		return fmt.Errorf("unknown command %q (want up, down, reset or status)", cmd)
	}

	version, err := migrator.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("get schema version: %w", err)
	}
	log.Info("done", "command", cmd, "schema_version", version)
	return nil
}

func logResults(log *slog.Logger, results []*goose.MigrationResult) {
	for _, r := range results {
		log.Info("migration applied",
			"version", r.Source.Version,
			"file", r.Source.Path,
			"direction", r.Direction,
			"duration", r.Duration.String(),
		)
	}
}
