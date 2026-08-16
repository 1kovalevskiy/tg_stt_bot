package logger

import (
	"log/slog"
	"strings"
)

// parseLogLevel maps the configured level name to a slog level,
// falling back to INFO for anything unknown.
func parseLogLevel(value string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
