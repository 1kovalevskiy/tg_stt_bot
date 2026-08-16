package app

import (
	adminController "github.com/1kovalevskiy/tg_stt_bot/internal/controllers/admin-controller"
	chatController "github.com/1kovalevskiy/tg_stt_bot/internal/controllers/chat-controller"
)

// initControllers creates the controllers, passing their dependencies as the
// consumer-side interfaces they declare.
func (a *App) initControllers() error {
	if a.Config == nil {
		return ErrNilConfig
	}

	if a.Providers.Telegram == nil {
		return ErrNilTelegramProvider
	}

	if a.Providers.STT == nil {
		return ErrNilSTTProvider
	}

	a.Controllers.Chat = chatController.NewController(a.Providers.Telegram, a.Providers.STT)
	a.Controllers.Admin = adminController.NewController(a.Providers.Telegram, a.Providers.STT, a.Config)

	return nil
}
