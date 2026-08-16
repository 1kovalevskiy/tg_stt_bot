// Package logger holds the logging infrastructure of the bot: the base stdout
// handler, the fan-out handler mirroring ERROR records into the Telegram
// service chat and the sink that delivers them from its own goroutine. The
// wiring only calls the constructors declared here; no other layer knows the
// service chat exists.
package logger

import "errors"

var (
	// ErrSinkDrainTimeout reports a service chat sink that was still
	// delivering queued records when the shutdown drain ran out of time.
	ErrSinkDrainTimeout = errors.New("service chat log sink drain timed out")
)
