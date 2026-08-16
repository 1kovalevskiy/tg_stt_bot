package telegram

import (
	"context"
	"errors"
	"fmt"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
)

// wrapRedactedError wraps err with the layer sentinel, replacing every occurrence
// of the bot token in the underlying error text: transport errors embed the
// request URL, which contains the token, and library errors may do the same.
// The original error is intentionally kept out of the chain so the token can
// never leak through a wrapped error.
//
// Flattening the text would also drop the cause, so a cancellation and a
// deadline are classified before the redaction and wrapped into the matching
// layer sentinel, which carries context.Canceled / context.DeadlineExceeded:
// the transport matches a shutdown with errors.Is and keeps it below ERROR,
// out of the service chat that is being drained at that very moment.
func (p *Provider) wrapRedactedError(sentinel, err error) error {
	msg := models.RedactToken(err.Error(), p.config.GetTelegramToken())

	switch {
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("%w: %w: %s", sentinel, providers.ErrTelegramRequestCanceled, msg)
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: %w: %s", sentinel, providers.ErrTelegramRequestTimeout, msg)
	default:
		return fmt.Errorf("%w: %s", sentinel, msg)
	}
}
