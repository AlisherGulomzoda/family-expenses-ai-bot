// Command api serves the REST API for the family head's web panel.
//
// Usage:
//
//	api              run the server
//	api healthcheck  exit 0 if the running server reports healthy
//	                 (for Docker healthchecks: the image has no curl)
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // the runtime image has no system time zone database

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/config"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/httpapi"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/logger"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/postgres"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

const shutdownTimeout = 10 * time.Second

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
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		return healthcheck(cfg.HTTPAddr)
	}

	log := logger.New(cfg.LogLevel, "api")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(ctx, cfg.DB.URL)
	if err != nil {
		return err
	}
	defer pool.Close()

	handler := httpapi.NewHandler(httpapi.Deps{
		Expenses: expense.NewService(postgres.NewExpenseRepo(pool), clock.System{}, cfg.Timezone.Location),
		Users:    user.NewService(postgres.NewUserRepo(pool)),
		DB:       pool,
		Log:      log,
	})
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("api started", "addr", cfg.HTTPAddr, "timezone", cfg.Timezone.String())
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	log.Info("api stopping")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	log.Info("api stopped")
	return nil
}

// healthcheck calls /healthz of the server listening on addr.
func healthcheck(addr string) error {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("parse HTTP_ADDR %q: %w", addr, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+port+"/healthz", http.NoBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check: status %d", resp.StatusCode)
	}
	return nil
}
