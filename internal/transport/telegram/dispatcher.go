// Package telegram implements the Telegram long polling transport: it builds
// the bot client, matches incoming updates, checks whether the bot serves
// their chat, maps them to the pure models the controllers accept and reports
// controller failures to the log. It holds no scenario logic of its own.
package telegram

import (
	"context"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/go-telegram/bot"
)

type (
	// chatHandler is the consumer-side interface of the chat controller.
	chatHandler interface {
		HandleVoice(ctx context.Context, audio models.IncomingAudio) error
		HandleVideoNote(ctx context.Context, audio models.IncomingAudio) error
	}

	// adminHandler is the consumer-side interface of the admin controller.
	adminHandler interface {
		HandleCommand(ctx context.Context, chatID int64, command string) error
	}

	// botRegistrar is the consumer-side interface of the bot client's handler
	// registration.
	botRegistrar interface {
		RegisterHandlerMatchFunc(matchFunc bot.MatchFunc, handler bot.HandlerFunc, middlewares ...bot.Middleware) string
	}

	// dispatcherConfig is the consumer-side interface of the application
	// config. The access rules are read on every update, never snapshot into
	// the dispatcher.
	dispatcherConfig interface {
		GetTelegramAdminID() int64
		GetTelegramAllowedChats() []int64
	}

	// Dispatcher routes Telegram updates to the controllers. Updates from
	// chats the bot does not serve are silently ignored. It holds no mutable
	// state and is safe for concurrent use by the update workers.
	Dispatcher struct {
		chat   chatHandler
		admin  adminHandler
		config dispatcherConfig
	}
)

// NewDispatcher creates a Dispatcher on top of the controllers it routes to
// and the application config holding the chat access rules.
func NewDispatcher(chat chatHandler, admin adminHandler, config dispatcherConfig) *Dispatcher {
	return &Dispatcher{
		chat:   chat,
		admin:  admin,
		config: config,
	}
}
