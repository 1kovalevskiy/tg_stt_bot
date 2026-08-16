package telegram

import (
	"fmt"
	"strings"
)

// redactedToken is what replaces the bot token in error text.
const redactedToken = "[REDACTED]"

// wrapRedacted wraps err with the layer sentinel, replacing every occurrence
// of the bot token in the underlying error text: transport errors embed the
// request URL, which contains the token, and library errors may do the same.
// The original error chain is intentionally dropped so the token can never
// leak through a wrapped error.
func (p *Provider) wrapRedacted(sentinel, err error) error {
	msg := err.Error()

	if token := p.config.GetTelegramToken(); token != "" {
		msg = strings.ReplaceAll(msg, token, redactedToken)
	}

	return fmt.Errorf("%w: %s", sentinel, msg)
}
