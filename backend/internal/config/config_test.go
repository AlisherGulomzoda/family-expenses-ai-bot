package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

// setEnv sets the given variables for the test and clears every other variable
// the config package reads, so tests do not depend on the developer's shell.
func setEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	known := []string{
		"LOG_LEVEL", "APP_TIMEZONE", "DATABASE_URL", "HTTP_ADDR",
		"TELEGRAM_BOT_TOKEN", "ALLOWED_TELEGRAM_IDS",
		"LLM_BASE_URL", "LLM_API_KEY", "LLM_MODEL", "LLM_TIMEOUT", "LLM_TEMPERATURE",
		"AGENT_HISTORY_LIMIT", "AGENT_MAX_STEPS", "MCP_TELEGRAM_ID",
	}
	for _, k := range known {
		t.Setenv(k, vars[k])
	}
}

func TestLoadAPIDefaults(t *testing.T) {
	setEnv(t, map[string]string{"DATABASE_URL": "postgres://localhost/db"})

	cfg, err := Load[API]()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.Timezone.String() != "Europe/Moscow" {
		t.Errorf("Timezone = %v, want Europe/Moscow", cfg.Timezone)
	}
}

func TestLoadBotDefaultsAndWhitelist(t *testing.T) {
	setEnv(t, map[string]string{
		"DATABASE_URL":         "postgres://localhost/db",
		"TELEGRAM_BOT_TOKEN":   "token",
		"LLM_API_KEY":          "key",
		"ALLOWED_TELEGRAM_IDS": "111,222",
		"LOG_LEVEL":            "debug",
	})

	cfg, err := Load[Bot]()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLM.BaseURL != "https://api.deepseek.com" || cfg.LLM.Model != "deepseek-chat" {
		t.Errorf("LLM defaults = %q / %q", cfg.LLM.BaseURL, cfg.LLM.Model)
	}
	if cfg.LLM.Timeout != 60*time.Second || cfg.LLM.Temperature != 0.1 {
		t.Errorf("LLM timeout/temperature = %v / %v", cfg.LLM.Timeout, cfg.LLM.Temperature)
	}
	if cfg.HistoryLimit != 20 || cfg.MaxSteps != 5 {
		t.Errorf("Agent = %+v", cfg.Agent)
	}
	if len(cfg.AllowedIDs) != 2 || cfg.AllowedIDs[0] != 111 || cfg.AllowedIDs[1] != 222 {
		t.Errorf("AllowedIDs = %v, want [111 222]", cfg.AllowedIDs)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want debug", cfg.LogLevel)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		load    func() error
		vars    map[string]string
		wantErr []string
	}{
		{
			name:    "api without database url",
			load:    func() error { _, err := Load[API](); return err },
			vars:    map[string]string{},
			wantErr: []string{"DATABASE_URL"},
		},
		{
			name:    "bot reports every missing variable",
			load:    func() error { _, err := Load[Bot](); return err },
			vars:    map[string]string{"DATABASE_URL": "postgres://localhost/db"},
			wantErr: []string{"TELEGRAM_BOT_TOKEN", "LLM_API_KEY"},
		},
		{
			name: "unknown time zone",
			load: func() error { _, err := Load[Migrate](); return err },
			vars: map[string]string{
				"DATABASE_URL": "postgres://localhost/db",
				"APP_TIMEZONE": "Mars/Olympus",
			},
			wantErr: []string{"Mars/Olympus"},
		},
		{
			name:    "temperature out of range",
			load:    func() error { _, err := Load[Eval](); return err },
			vars:    map[string]string{"LLM_API_KEY": "key", "LLM_TEMPERATURE": "3"},
			wantErr: []string{"LLM_TEMPERATURE"},
		},
		{
			name: "invalid whitelist id",
			load: func() error { _, err := Load[Bot](); return err },
			vars: map[string]string{
				"DATABASE_URL":         "postgres://localhost/db",
				"TELEGRAM_BOT_TOKEN":   "token",
				"LLM_API_KEY":          "key",
				"ALLOWED_TELEGRAM_IDS": "111,abc",
			},
			// Parse errors name the struct field and the bad value.
			wantErr: []string{"AllowedIDs", "abc"},
		},
		{
			name: "zero agent steps",
			load: func() error { _, err := Load[Bot](); return err },
			vars: map[string]string{
				"DATABASE_URL":       "postgres://localhost/db",
				"TELEGRAM_BOT_TOKEN": "token",
				"LLM_API_KEY":        "key",
				"AGENT_MAX_STEPS":    "0",
			},
			wantErr: []string{"AGENT_MAX_STEPS"},
		},
		{
			name:    "mcp without a user",
			load:    func() error { _, err := Load[MCP](); return err },
			vars:    map[string]string{"DATABASE_URL": "postgres://localhost/db"},
			wantErr: []string{"MCP_TELEGRAM_ID"},
		},
		{
			name:    "mcp with a negative user id",
			load:    func() error { _, err := Load[MCP](); return err },
			vars:    map[string]string{"DATABASE_URL": "postgres://localhost/db", "MCP_TELEGRAM_ID": "-1"},
			wantErr: []string{"MCP_TELEGRAM_ID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.vars)
			err := tt.load()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
		})
	}
}
