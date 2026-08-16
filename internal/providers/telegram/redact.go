package telegram

import (
	"fmt"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

// wrapRedacted wraps err with the layer sentinel, replacing every occurrence
// of the bot token in the underlying error text: transport errors embed the
// request URL, which contains the token, and library errors may do the same.
// The original error chain is intentionally dropped so the token can never
// leak through a wrapped error.
func (p *Provider) wrapRedacted(sentinel, err error) error {
	msg := models.RedactToken(err.Error(), p.config.GetTelegramToken())

	return fmt.Errorf("%w: %s", sentinel, msg)
}
