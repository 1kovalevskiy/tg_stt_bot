package app

import (
	"log/slog"

	"github.com/1kovalevskiy/tg_stt_bot/internal/logger"
)

// initLogs installs the base JSON logger writing to stdout.
func (a *App) initLogs() error {
	if a.Config == nil {
		return ErrNilConfig
	}

	handler := logger.NewBaseHandler(a.Config)

	a.logHandler = handler
	slog.SetDefault(slog.New(handler))

	return nil
}

// initLogSink starts the service chat sink and wraps the base logger with the
// handler mirroring ERROR records into it. It runs after the providers because
// the sink delivers the records through the Telegram provider.
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

	sink := logger.NewServiceChatSink(a.Providers.Telegram, a.Config)

	a.addCloser("service-chat-log-sink", sink.Close)
	slog.SetDefault(slog.New(logger.NewServiceChatHandler(a.logHandler, sink)))

	return nil
}
