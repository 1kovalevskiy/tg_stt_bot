package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestNewBaseHandler_TakesTheLevelFromTheConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		level       string
		record      slog.Level
		wantEnabled bool
	}{
		{name: "debug config takes debug", level: "DEBUG", record: slog.LevelDebug, wantEnabled: true},
		{name: "info config drops debug", level: "INFO", record: slog.LevelDebug, wantEnabled: false},
		{name: "info config takes error", level: "INFO", record: slog.LevelError, wantEnabled: true},
		{name: "error config drops warn", level: "ERROR", record: slog.LevelWarn, wantEnabled: false},
		{name: "unknown config behaves as info", level: "verbose", record: slog.LevelInfo, wantEnabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := NewBaseHandler(fakeConfig{level: tt.level})

			if got := handler.Enabled(context.Background(), tt.record); got != tt.wantEnabled {
				t.Errorf("Enabled(%v) = %v for level %q, want %v", tt.record, got, tt.level, tt.wantEnabled)
			}
		})
	}
}
