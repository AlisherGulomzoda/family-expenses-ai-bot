// Command mcp serves the expense tools over the Model Context Protocol (stdio)
// on behalf of one family member (MCP_TELEGRAM_ID).
//
// stdout carries the protocol, so logs go to stderr.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"syscall"
	_ "time/tzdata" // the runtime image has no system time zone database

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/agent/tools"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/clock"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/config"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/expense"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/logger"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/mcpserver"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/postgres"
	"github.com/AlisherGulomzoda/family-expenses-ai-bot/backend/internal/user"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "mcp:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load[config.MCP]()
	if err != nil {
		return err
	}
	log := logger.NewWithWriter(os.Stderr, cfg.LogLevel, "mcp")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(ctx, cfg.DB.URL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Expenses reference users, and the name tells the client whose data it
	// sees: the member must have written to the bot at least once.
	members, err := user.NewService(postgres.NewUserRepo(pool)).List(ctx)
	if err != nil {
		return fmt.Errorf("load family members: %w", err)
	}
	i := slices.IndexFunc(members, func(u user.User) bool { return u.TelegramID == cfg.TelegramID })
	if i < 0 {
		return fmt.Errorf("MCP_TELEGRAM_ID %d is not a family member yet: write to the bot once first", cfg.TelegramID)
	}
	member := members[i]

	expenses := expense.NewService(postgres.NewExpenseRepo(pool), clock.System{}, cfg.Timezone.Location)
	server := mcpserver.New(tools.Default(), expenses.ForUser(member.TelegramID), member.Name, log)

	log.Info("mcp server starting", "transport", "stdio", "user", member.Name, "timezone", cfg.Timezone.String())
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil && ctx.Err() == nil {
		return err
	}
	log.Info("mcp server stopped")
	return nil
}
