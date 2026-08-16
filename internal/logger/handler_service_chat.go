package logger

import (
	"io"
	"log/slog"
)

// NewServiceChatHandler wraps the base handler with a fan-out that also
// mirrors ERROR records into the sink. Rendering is left to the standard text
// handler: the sink has no renderer of its own and must not grow one.
func NewServiceChatHandler(base slog.Handler, sink io.Writer) slog.Handler {
	mirror := slog.NewTextHandler(sink, &slog.HandlerOptions{
		Level:       slog.LevelError,
		ReplaceAttr: dropRecordTime,
	})

	return newFanOutHandler(base, mirror)
}

// dropRecordTime removes the record timestamp from the mirrored text:
// Telegram stamps every message on its own.
func dropRecordTime(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) == 0 && attr.Key == slog.TimeKey {
		return slog.Attr{}
	}

	return attr
}
