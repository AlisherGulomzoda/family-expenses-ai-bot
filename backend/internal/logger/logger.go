// Package logger builds the structured JSON logger used by all binaries.
package logger

import (
	"io"
	"log/slog"
	"os"
)

// New returns a JSON logger writing to stdout. Every record carries
// the service name so logs of api and bot can be told apart.
func New(level slog.Level, service string) *slog.Logger {
	return NewWithWriter(os.Stdout, level, service)
}

// NewWithWriter is like New but writes to w.
func NewWithWriter(w io.Writer, level slog.Level, service string) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(h).With(slog.String("service", service))
}
