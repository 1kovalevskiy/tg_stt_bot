package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

type (
	// serviceChatSender is the consumer-side interface used to deliver
	// records to the service chat.
	serviceChatSender interface {
		SendMessage(ctx context.Context, chatID int64, text string) error
	}

	// serviceChatSink delivers formatted log records to the service chat from
	// its own goroutine, so logging never blocks on the Telegram API.
	serviceChatSink struct {
		sender serviceChatSender
		chatID int64
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

	// fanOutHandler writes every record to all the handlers it wraps: the base
	// stdout handler and the ERROR-only handler mirroring into the service chat.
	fanOutHandler struct {
		handlers []slog.Handler
	}
)

// newServiceChatSink starts a sink delivering records to chatID. A nil errLog
// falls back to a stderr logger.
func newServiceChatSink(sender serviceChatSender, chatID int64, queueSize int, errLog *log.Logger) *serviceChatSink {
	if errLog == nil {
		errLog = log.New(os.Stderr, "service-chat-log: ", log.LstdFlags)
	}

	if queueSize < 1 {
		queueSize = 1
	}

	sink := &serviceChatSink{
		sender:       sender,
		chatID:       chatID,
		queue:        make(chan string, queueSize),
		done:         make(chan struct{}),
		errLog:       errLog,
		drainTimeout: models.ServiceChatDrainTimeout,
	}

	go sink.run()

	return sink
}

// Write makes the sink an io.Writer, so a standard slog handler can render the
// records mirrored to the service chat. It never fails: a delivery problem is
// reported to the sink's own error log, not back to the logging caller.
func (s *serviceChatSink) Write(record []byte) (int, error) {
	text := strings.TrimRight(string(record), "\n")

	s.send(models.TruncateText(text, models.TelegramMessageLimit))

	return len(record), nil
}

// send queues text for delivery without ever blocking the caller: an overflow
// or a closed sink drops the record and reports it to stderr.
func (s *serviceChatSink) send(text string) {
	if text == "" {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		s.errLog.Printf("dropped record, sink is closed: %s", text)

		return
	}

	select {
	case s.queue <- text:
	default:
		s.errLog.Printf("dropped record, queue is full: %s", text)
	}
}

// Close stops accepting records and waits for the queue to drain.
func (s *serviceChatSink) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()

		return nil
	}

	s.closed = true
	close(s.queue)
	s.mu.Unlock()

	select {
	case <-s.done:
		return nil
	case <-time.After(s.drainTimeout):
		return ErrLogSinkDrainTimeout
	}
}

// run delivers queued records until the queue is closed and drained, dropping
// everything above the rate limit.
func (s *serviceChatSink) run() {
	defer close(s.done)

	for text := range s.queue {
		s.rotateWindow(time.Now())

		if s.windowSent >= models.ServiceChatRateBurst {
			s.suppressed++

			continue
		}

		s.windowSent++
		s.deliver(text)
	}

	s.reportSuppressed()
}

// rotateWindow starts a new rate window once the current one has expired,
// reporting what the expired one had to drop.
func (s *serviceChatSink) rotateWindow(now time.Time) {
	if !s.windowStart.IsZero() && now.Sub(s.windowStart) < models.ServiceChatRateWindow {
		return
	}

	s.reportSuppressed()

	s.windowStart = now
	s.windowSent = 0
}

// reportSuppressed sends a summary of the records the rate limit dropped. The
// summary itself bypasses the limit: it is at most one message per window and
// is the only trace those records leave in the chat.
func (s *serviceChatSink) reportSuppressed() {
	if s.suppressed == 0 {
		return
	}

	count := s.suppressed
	s.suppressed = 0

	s.deliver(fmt.Sprintf(models.MsgSuppressedFormat, count))
}

// deliver sends a single record to the service chat.
func (s *serviceChatSink) deliver(text string) {
	ctx, cancel := context.WithTimeout(context.Background(), models.ServiceChatSendTimeout)
	defer cancel()

	if err := s.sender.SendMessage(ctx, s.chatID, text); err != nil {
		s.errLog.Printf("failed to send record: %v", err)
	}
}

// newServiceChatMirror renders ERROR records as plain text into the sink.
// Rendering is left to the standard text handler; only the timestamp is
// dropped, because Telegram stamps every message on its own.
func newServiceChatMirror(sink *serviceChatSink) slog.Handler {
	return slog.NewTextHandler(sink, &slog.HandlerOptions{
		Level: slog.LevelError,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if len(groups) == 0 && attr.Key == slog.TimeKey {
				return slog.Attr{}
			}

			return attr
		},
	})
}

// newFanOutHandler builds a handler writing every record to all handlers.
func newFanOutHandler(handlers ...slog.Handler) *fanOutHandler {
	return &fanOutHandler{handlers: handlers}
}

// Enabled reports whether any of the wrapped handlers takes the level: the
// service chat handler accepts ERROR even when stdout is configured higher.
func (h *fanOutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

// Handle writes the record to every wrapped handler that takes its level.
func (h *fanOutHandler) Handle(ctx context.Context, record slog.Record) error {
	var handleErr error

	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, record.Level) {
			continue
		}

		if err := handler.Handle(ctx, record.Clone()); err != nil {
			handleErr = errors.Join(handleErr, err)
		}
	}

	return handleErr
}

// WithAttrs delegates to every wrapped handler.
func (h *fanOutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	return newFanOutHandler(h.derive(func(handler slog.Handler) slog.Handler {
		return handler.WithAttrs(attrs)
	})...)
}

// WithGroup delegates to every wrapped handler.
func (h *fanOutHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return newFanOutHandler(h.derive(func(handler slog.Handler) slog.Handler {
		return handler.WithGroup(name)
	})...)
}

// derive applies fn to every wrapped handler.
func (h *fanOutHandler) derive(fn func(slog.Handler) slog.Handler) []slog.Handler {
	derived := make([]slog.Handler, 0, len(h.handlers))

	for _, handler := range h.handlers {
		derived = append(derived, fn(handler))
	}

	return derived
}
