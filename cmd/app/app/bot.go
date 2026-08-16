package app

import (
	telegramTransport "github.com/1kovalevskiy/tg_stt_bot/internal/transport/telegram"
)

// initBotClient creates the long polling client. It runs before the providers
// because the Telegram provider is built on top of this client.
func (a *App) initBotClient() error {
	if a.Config == nil {
		return ErrNilConfig
	}

	if a.logHandler == nil {
		return ErrNilLogHandler
	}

	client, err := telegramTransport.NewBotClient(a.Config, a.logHandler)
	if err != nil {
		return err
	}

	a.Bot = client

	return nil
}

// initBotHandlers registers the update dispatcher on the bot client. It runs
// after the controllers the dispatcher routes to are created.
func (a *App) initBotHandlers() error {
	if a.Config == nil {
		return ErrNilConfig
	}

	if a.Bot == nil {
		return ErrNilBot
	}

	if a.Controllers.Chat == nil || a.Controllers.Admin == nil {
		return ErrNilController
	}

	dispatcher := telegramTransport.NewDispatcher(a.Controllers.Chat, a.Controllers.Admin, a.Config)
	dispatcher.RegisterHandlers(a.Bot)

	return nil
}
