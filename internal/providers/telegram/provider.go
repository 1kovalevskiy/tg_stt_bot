// Package telegram provides a Telegram Bot API provider: downloading files
// and sending messages/replies. Errors returned by this package never contain
// the bot token.
package telegram

import (
	"context"
	"net/http"

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

	// Provider is a Telegram Bot API client. It holds no mutable state
	// and is safe for concurrent use.
	Provider struct {
		api    botAPI
		token  string
		client httpDoer
	}
)

// NewProvider creates a Provider on top of the bot API client. The token is
// needed to build file download URLs and to redact errors.
func NewProvider(api botAPI, token string, client httpDoer) *Provider {
	return &Provider{
		api:    api,
		token:  token,
		client: client,
	}
}
