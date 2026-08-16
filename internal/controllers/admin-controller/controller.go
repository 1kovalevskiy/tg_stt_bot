// Package adminController implements the service commands the bot admin can
// run in a private chat: /status checks the STT service and /chats prints the
// configured chat whitelist.
package adminController

import "context"

type (
	// telegramProvider is the consumer-side interface of the Telegram provider.
	telegramProvider interface {
		SendMessage(ctx context.Context, chatID int64, text string) error
	}

	// sttProvider is the consumer-side interface of the STT provider.
	sttProvider interface {
		Health(ctx context.Context) (string, error)
	}

	// configProvider is the consumer-side interface of the application config.
	configProvider interface {
		GetTelegramAllowedChats() []int64
	}

	// Controller handles admin commands. It holds no mutable state and is safe
	// for concurrent use as long as its dependencies are.
	Controller struct {
		telegram telegramProvider
		stt      sttProvider
		config   configProvider
	}
)

// NewController creates a Controller on top of the Telegram and STT providers
// and the application config.
func NewController(telegram telegramProvider, stt sttProvider, config configProvider) *Controller {
	return &Controller{
		telegram: telegram,
		stt:      stt,
		config:   config,
	}
}
