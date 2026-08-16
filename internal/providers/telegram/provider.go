// Package telegram provides a Telegram Bot API provider: downloading files
// and sending messages/replies. Errors returned by this package never contain
// the bot token.
package telegram

import (
	"context"
	"net/http"
	"time"

	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// fileBaseURL is the Bot API host used to download files by file_path.
const fileBaseURL = "https://api.telegram.org"

type (
	// botAPI is the consumer-side interface of *bot.Bot with the only
	// methods this provider needs.
	botAPI interface {
		GetFile(ctx context.Context, params *bot.GetFileParams) (*tgmodels.File, error)
		SendMessage(ctx context.Context, params *bot.SendMessageParams) (*tgmodels.Message, error)
	}

	// httpDoer is the consumer-side interface of an HTTP client used to
	// download files.
	httpDoer interface {
		Do(req *http.Request) (*http.Response, error)
	}

	// configProvider is the consumer-side interface of the application config.
	// The token is needed to build file download URLs and to redact errors;
	// the timeouts bound a single Bot API call and a single file download.
	configProvider interface {
		GetTelegramToken() string
		GetTelegramAPITimeout() time.Duration
		GetTelegramDownloadTimeout() time.Duration
	}

	// Provider is a Telegram Bot API client. It holds no mutable state
	// and is safe for concurrent use.
	Provider struct {
		api    botAPI
		config configProvider
		client httpDoer
	}
)

// NewProvider creates a Provider on top of the bot API client. The settings
// are read from the config on every call, never snapshot into the provider.
func NewProvider(api botAPI, config configProvider, client httpDoer) *Provider {
	return &Provider{
		api:    api,
		config: config,
		client: client,
	}
}
