// Package transport holds the sentinel errors of the transport layer. Every
// transport of the layer (currently the Telegram long polling client) wraps
// its failures into these sentinels, so the wiring can match them with
// errors.Is. Names are prefixed with the external service the error belongs
// to, exactly like in the provider layer: a second transport would otherwise
// share a sentinel with the first one.
package transport

import "errors"

var (
	// ErrTelegramCreateBotClient reports a long polling client the library
	// refused to build: an empty token, or a token the Bot API rejected on
	// the startup getMe call. The token never appears in the wrapped text.
	ErrTelegramCreateBotClient = errors.New("failed to create telegram bot client")
)
