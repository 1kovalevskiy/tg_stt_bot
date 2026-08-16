package logger

import (
	"log/slog"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  slog.Level
	}{
		{name: "debug", value: "DEBUG", want: slog.LevelDebug},
		{name: "lowercase info", value: "info", want: slog.LevelInfo},
		{name: "warn", value: "WARN", want: slog.LevelWarn},
		{name: "warning", value: " warning ", want: slog.LevelWarn},
		{name: "error", value: "ERROR", want: slog.LevelError},
		{name: "unknown falls back to info", value: "verbose", want: slog.LevelInfo},
		{name: "empty falls back to info", value: "", want: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := parseLogLevel(tt.value); got != tt.want {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
