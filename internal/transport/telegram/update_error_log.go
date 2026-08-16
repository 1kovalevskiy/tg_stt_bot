package telegram

import (
	"context"
	"errors"
	"log/slog"
)

// logUpdateError reports a controller failure. A canceled context means the
// bot is shutting down rather than something being broken, so it stays below
// ERROR and out of the service chat, which is being closed at that moment.
func logUpdateError(msg string, err error, args ...any) {
	args = append([]any{"err", err}, args...)

	if errors.Is(err, context.Canceled) {
		slog.Warn(msg+" (shutting down)", args...)

		return
	}

	slog.Error(msg, args...)
}
