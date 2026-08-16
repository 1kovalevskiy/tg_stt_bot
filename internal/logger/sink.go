package logger

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

type (
	// messageSender is the consumer-side interface used to deliver records to
	// the service chat.
	messageSender interface {
		SendMessage(ctx context.Context, chatID int64, text string) error
	}

	// sinkConfig is the consumer-side interface of the application config.
	// The chat id is read on every delivery, never snapshot into the sink.
	sinkConfig interface {
		GetTelegramServiceChatID() int64
	}

	// ServiceChatSink delivers formatted log records to the service chat from
	// its own goroutine, so logging never blocks on the Telegram API.
	ServiceChatSink struct {
		sender messageSender
		config sinkConfig
		queue  chan string
		done   chan struct{}
		// errLog reports the sink's own failures. It is a plain logger on
		// purpose: routing them through slog would feed them back into this
		// sink and recurse.
		errLog *log.Logger
		// drainTimeout bounds the drain in Close. It is a field so tests can
		// drive the timeout branch without waiting out a real send.
		drainTimeout time.Duration

		mu     sync.RWMutex
		closed bool

		// Rate limiting state, owned by the run goroutine alone.
		windowStart time.Time
		windowSent  int
		suppressed  int
	}
)

// NewServiceChatSink starts a sink delivering records to the configured
// service chat. The returned sink is an io.Writer for the mirror handler and
// has to be closed on shutdown to drain what is still queued.
func NewServiceChatSink(sender messageSender, config sinkConfig) *ServiceChatSink {
	return newServiceChatSink(sender, config, models.ServiceChatQueueSize, nil)
}

// newServiceChatSink starts a sink with an explicit queue size and error log.
// A nil errLog falls back to a stderr logger.
func newServiceChatSink(
	sender messageSender, config sinkConfig, queueSize int, errLog *log.Logger,
) *ServiceChatSink {
	if errLog == nil {
		errLog = log.New(os.Stderr, "service-chat-log: ", log.LstdFlags)
	}

	if queueSize < 1 {
		queueSize = 1
	}

	sink := &ServiceChatSink{
		sender:       sender,
		config:       config,
		queue:        make(chan string, queueSize),
		done:         make(chan struct{}),
		errLog:       errLog,
		drainTimeout: models.ServiceChatDrainTimeout,
	}

	go sink.runQueue()

	return sink
}
