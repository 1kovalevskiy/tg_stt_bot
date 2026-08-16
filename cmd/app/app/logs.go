package app

import (
	"log/slog"
	"os"
	"strings"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

// initLogs installs the base JSON logger writing to stdout.
func (a *App) initLogs() error {
	if a.Config == nil {
		return ErrNilConfig
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(a.Config.GetAppLogLevel()),
	})

	a.logHandler = handler
	slog.SetDefault(slog.New(handler))

	return nil
}

// initLogSink wraps the base logger with a fan-out handler that mirrors ERROR
// records to the Telegram service chat. It runs after the providers because
// it needs the Telegram provider to deliver the records.
func (a *App) initLogSink() error {
	if a.Config == nil {
		return ErrNilConfig
	}

	if a.logHandler == nil {
		return ErrNilLogHandler
	}

	if a.Providers.Telegram == nil {
		return ErrNilTelegramProvider
	}

	sink := newServiceChatSink(
		a.Providers.Telegram,
		a.Config.GetTelegramServiceChatID(),
		models.ServiceChatQueueSize,
		nil,
	)

	a.addCloser("service-chat-log-sink", sink.Close)
	slog.SetDefault(slog.New(newFanOutHandler(a.logHandler, newServiceChatMirror(sink))))

	return nil
}

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
