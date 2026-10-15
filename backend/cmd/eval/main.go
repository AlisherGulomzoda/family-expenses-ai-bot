// Command eval runs the agent evaluation suite against a real LLM.
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
		fmt.Fprintln(os.Stderr, "eval:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load[config.Eval]()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, "eval")

	// The evaluation suite is added in BE-11.
	log.Info("no eval cases yet", "llm_model", cfg.LLM.Model)
	return nil
}
