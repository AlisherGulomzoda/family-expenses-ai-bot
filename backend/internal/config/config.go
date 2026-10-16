// Package config loads process configuration from environment variables.
//
// Every binary has its own config struct, so a process only requires the
// variables it actually uses (the API server does not need a bot token).
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/caarlos0/env/v11"
)

// Common holds settings shared by all binaries.
type Common struct {
	LogLevel slog.Level `env:"LOG_LEVEL"    envDefault:"info"`
	Timezone Location   `env:"APP_TIMEZONE" envDefault:"Europe/Moscow"`
}

// DB holds database connection settings.
type DB struct {
	URL string `env:"DATABASE_URL,required,notEmpty"`
}

// Telegram holds Telegram bot settings.
type Telegram struct {
	Token string `env:"TELEGRAM_BOT_TOKEN,required,notEmpty"`
	// AllowedIDs is the family whitelist. Empty means everyone is allowed.
	AllowedIDs []int64 `env:"ALLOWED_TELEGRAM_IDS" envSeparator:","`
}

// LLM holds settings of an OpenAI-compatible chat completions provider.
type LLM struct {
	BaseURL     string        `env:"LLM_BASE_URL"    envDefault:"https://api.deepseek.com"`
	APIKey      string        `env:"LLM_API_KEY,required,notEmpty"`
	Model       string        `env:"LLM_MODEL"       envDefault:"deepseek-chat"`
	Timeout     time.Duration `env:"LLM_TIMEOUT"     envDefault:"60s"`
	Temperature float64       `env:"LLM_TEMPERATURE" envDefault:"0.1"`
}

func (c LLM) validate() error {
	var errs []error
	if c.Timeout <= 0 {
		errs = append(errs, errors.New("LLM_TIMEOUT must be positive"))
	}
	if c.Temperature < 0 || c.Temperature > 2 {
		errs = append(errs, errors.New("LLM_TEMPERATURE must be between 0 and 2"))
	}
	return errors.Join(errs...)
}

// Agent holds settings of the expense agent loop.
type Agent struct {
	HistoryLimit int `env:"AGENT_HISTORY_LIMIT" envDefault:"20"`
	MaxSteps     int `env:"AGENT_MAX_STEPS"     envDefault:"5"`
}

func (c Agent) validate() error {
	var errs []error
	if c.HistoryLimit < 0 {
		errs = append(errs, errors.New("AGENT_HISTORY_LIMIT must not be negative"))
	}
	if c.MaxSteps < 1 {
		errs = append(errs, errors.New("AGENT_MAX_STEPS must be at least 1"))
	}
	return errors.Join(errs...)
}

// API is the configuration of the REST API server.
type API struct {
	Common
	DB
	HTTPAddr string `env:"HTTP_ADDR" envDefault:":8080"`
}

// Bot is the configuration of the Telegram bot.
type Bot struct {
	Common
	DB
	Telegram
	LLM
	Agent
}

func (c Bot) validate() error {
	return errors.Join(c.LLM.validate(), c.Agent.validate())
}

// Migrate is the configuration of the migration tool.
type Migrate struct {
	Common
	DB
}

// Eval is the configuration of the agent evaluation tool.
// It runs on in-memory storage, so it needs no database.
type Eval struct {
	Common
	LLM
}

func (c Eval) validate() error {
	return c.LLM.validate()
}

// MCP is the configuration of the MCP server. It acts on behalf of one family
// member, like a Telegram chat with the bot.
type MCP struct {
	Common
	DB
	TelegramID int64 `env:"MCP_TELEGRAM_ID,required,notEmpty"`
}

func (c MCP) validate() error {
	if c.TelegramID <= 0 {
		return errors.New("MCP_TELEGRAM_ID must be a positive Telegram user ID")
	}
	return nil
}

type validator interface {
	validate() error
}

// Load parses environment variables into a config of type T and validates it.
func Load[T API | Bot | Migrate | Eval | MCP]() (T, error) {
	var cfg T
	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if v, ok := any(cfg).(validator); ok {
		if err := v.validate(); err != nil {
			return cfg, fmt.Errorf("invalid config: %w", err)
		}
	}
	return cfg, nil
}

// Location is a time zone that can be parsed from an IANA name.
type Location struct {
	*time.Location
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (l *Location) UnmarshalText(text []byte) error {
	loc, err := time.LoadLocation(string(text))
	if err != nil {
		return fmt.Errorf("unknown time zone %q: %w", text, err)
	}
	l.Location = loc
	return nil
}
