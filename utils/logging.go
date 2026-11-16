package utils

import (
	"io"
	"log/slog"
)

// NewJSONLogger writes one JSON event per line (time, level, msg, fields).
// cmd/main.go installs it with slog.SetDefault, which also routes every
// log.Printf in the code base through it, so all server logs are JSON.
func NewJSONLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, nil))
}
