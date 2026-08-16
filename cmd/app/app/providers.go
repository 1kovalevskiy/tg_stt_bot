package app

import (
	"net/http"
	"time"

	sttProvider "github.com/1kovalevskiy/tg_stt_bot/internal/providers/stt"
	telegramProvider "github.com/1kovalevskiy/tg_stt_bot/internal/providers/telegram"
)

// fileDownloadTimeout bounds a single Telegram file download. Files are
// capped at 20 MB by the Bot API, so this only guards against a stuck
// connection.
const fileDownloadTimeout = 2 * time.Minute

// initProviders creates the concrete providers. This is the only place that
// knows their implementations.
func (a *App) initProviders() error {
	if a.Config == nil {
		return ErrNilConfig
	}

	if a.Bot == nil {
		return ErrNilBot
	}

	a.Providers.STT = sttProvider.NewProvider(
		a.Config.GetSTTBaseURL(),
		a.Config.GetSTTLanguage(),
		&http.Client{Timeout: a.Config.GetSTTTimeout()},
	)

	a.Providers.Telegram = telegramProvider.NewProvider(
		a.Bot,
		a.Config.GetTelegramToken(),
		&http.Client{Timeout: fileDownloadTimeout},
	)

	return nil
}
