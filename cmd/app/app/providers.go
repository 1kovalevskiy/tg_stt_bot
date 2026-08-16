package app

import (
	"net/http"

	sttProvider "github.com/1kovalevskiy/tg_stt_bot/internal/providers/stt"
	telegramProvider "github.com/1kovalevskiy/tg_stt_bot/internal/providers/telegram"
)

// initProviders creates the concrete providers. This is the only place that
// knows their implementations.
//
// The HTTP clients carry no timeout on purpose: every provider method derives
// a child context with the timeout from the config, so the context is the
// single source of truth for deadlines and there is no second, invisible one.
func (a *App) initProviders() error {
	if a.Config == nil {
		return ErrNilConfig
	}

	if a.Bot == nil {
		return ErrNilBot
	}

	a.Providers.STT = sttProvider.NewProvider(a.Config, &http.Client{})

	a.Providers.Telegram = telegramProvider.NewProvider(a.Bot, a.Config, &http.Client{})

	return nil
}
