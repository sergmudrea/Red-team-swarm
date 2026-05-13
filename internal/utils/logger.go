// Package utils provides small helper functions shared across the project.
package utils

import (
	"io"
	"log/slog"
	"os"
)

// NewLogger creates a structured logger that writes to the given io.Writer.
// If w is nil, os.Stdout is used.
func NewLogger(level slog.Level, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewJSONHandler(w, opts)
	return slog.New(handler)
}
