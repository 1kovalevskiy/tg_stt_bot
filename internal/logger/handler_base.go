package logger

import (
	"log/slog"
	"os"
)

// levelConfig is the consumer-side interface of the application config: the
// base handler only needs the configured log level.
type levelConfig interface {
	GetAppLogLevel() string
}

// NewBaseHandler builds the JSON handler writing every record to stdout at the
// configured level. It is also the handler the bot library reports its own
// failures to, so nothing it logs can reach the service chat sink.
func NewBaseHandler(config levelConfig) slog.Handler {
	return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(config.GetAppLogLevel()),
	})
}
