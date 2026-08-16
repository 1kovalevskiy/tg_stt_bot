package telegram

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/1kovalevskiy/tg_stt_bot/internal/transport"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// botClientConfig is the consumer-side interface of the application config:
// the client only needs the bot token, and only through its getter.
type botClientConfig interface {
	GetTelegramToken() string
}

// NewBotClient creates the long polling client the providers and the update
// handlers are built on.
//
// The library reports its own errors from inside sendMessage as well, and the
// service chat sink sends through sendMessage: routing them through the
// default logger would feed the sink from its own delivery path. They go to
// the base handler (stdout) only.
func NewBotClient(config botClientConfig, baseLogHandler slog.Handler) (*bot.Bot, error) {
	token := config.GetTelegramToken()
	botLog := slog.New(baseLogHandler)

	client, err := bot.New(
		token,
		bot.WithWorkers(models.BotWorkers),
		// Handlers run in the workers instead of a goroutine per update, so
		// models.BotWorkers actually caps the number of concurrent
		// transcriptions and the memory they hold.
		bot.WithNotAsyncHandlers(),
		bot.WithDefaultHandler(ignoreUpdate),
		bot.WithErrorsHandler(func(err error) {
			botLog.Error("telegram bot error", "err", models.RedactToken(err.Error(), token))
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s",
			transport.ErrTelegramCreateBotClient, models.RedactToken(err.Error(), token))
	}

	return client, nil
}

// ignoreUpdate drops updates no handler matched. The library default prints
// every such update, which is both noisy and leaks message content.
func ignoreUpdate(_ context.Context, _ *bot.Bot, _ *tgmodels.Update) {}
