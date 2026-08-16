package logger

import (
	"context"
	"errors"
	"log/slog"
)

// fanOutHandler writes every record to all the handlers it wraps: the base
// stdout handler and the ERROR-only handler mirroring into the service chat.
type fanOutHandler struct {
	handlers []slog.Handler
}

// newFanOutHandler builds a handler writing every record to all handlers.
func newFanOutHandler(handlers ...slog.Handler) *fanOutHandler {
	return &fanOutHandler{handlers: handlers}
}

// Enabled reports whether any of the wrapped handlers takes the level: the
// service chat handler accepts ERROR even when stdout is configured higher.
func (h *fanOutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

// Handle writes the record to every wrapped handler that takes its level.
func (h *fanOutHandler) Handle(ctx context.Context, record slog.Record) error {
	var handleErr error

	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, record.Level) {
			continue
		}

		if err := handler.Handle(ctx, record.Clone()); err != nil {
			handleErr = errors.Join(handleErr, err)
		}
	}

	return handleErr
}

// WithAttrs delegates to every wrapped handler.
func (h *fanOutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	return newFanOutHandler(h.deriveHandlers(func(handler slog.Handler) slog.Handler {
		return handler.WithAttrs(attrs)
	})...)
}

// WithGroup delegates to every wrapped handler.
func (h *fanOutHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return newFanOutHandler(h.deriveHandlers(func(handler slog.Handler) slog.Handler {
		return handler.WithGroup(name)
	})...)
}

// deriveHandlers applies fn to every wrapped handler.
func (h *fanOutHandler) deriveHandlers(fn func(slog.Handler) slog.Handler) []slog.Handler {
	derived := make([]slog.Handler, 0, len(h.handlers))

	for _, handler := range h.handlers {
		derived = append(derived, fn(handler))
	}

	return derived
}
