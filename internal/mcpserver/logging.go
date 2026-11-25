package mcpserver

import (
	"io"
	"log/slog"
	"os"
)

// LogOptions configures a logger built by NewLogger. A supplied Logger wins
// unless Writer is also set; Level and JSON only affect loggers built here.
type LogOptions struct {
	Writer io.Writer
	Logger *slog.Logger
	Level  slog.Leveler
	JSON   bool
}

// NewLogger builds the standard structured logger used by the core. It uses
// only log/slog from the standard library, so a consumer with mcp.NewServer can
// adopt the same logging behaviour without importing internal/app.
func NewLogger(o LogOptions) *slog.Logger {
	if o.Logger != nil && o.Writer == nil {
		return o.Logger
	}
	level := o.Level
	if level == nil {
		level = slog.LevelInfo
	}
	handlerOptions := &slog.HandlerOptions{Level: level}
	newHandler := func(w io.Writer) slog.Handler {
		if o.JSON {
			return slog.NewJSONHandler(w, handlerOptions)
		}
		return slog.NewTextHandler(w, handlerOptions)
	}
	if o.Writer != nil {
		return slog.New(newHandler(o.Writer))
	}
	return slog.New(newHandler(os.Stderr))
}
