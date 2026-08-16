package app

import (
	"context"
	"log"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

const (
	// serviceChatQueueSize is how many ERROR records may wait for delivery
	// before new ones are dropped instead of blocking the logging caller.
	serviceChatQueueSize = 64
	// serviceChatSendTimeout bounds a single delivery attempt.
	serviceChatSendTimeout = 15 * time.Second
	// serviceChatDrainTimeout bounds the queue drain on shutdown.
	serviceChatDrainTimeout = 5 * time.Second
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

		mu     sync.RWMutex
		closed bool
	}

	// serviceChatHandler is a slog handler that writes every record to the
	// wrapped handler and additionally mirrors ERROR records to the sink.
	serviceChatHandler struct {
		inner  slog.Handler
		sink   *serviceChatSink
		attrs  []string
		groups []string
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
		sender: sender,
		chatID: chatID,
		queue:  make(chan string, queueSize),
		done:   make(chan struct{}),
		errLog: errLog,
	}

	go sink.run()

	return sink
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
	case <-time.After(serviceChatDrainTimeout):
		return ErrLogSinkDrainTimeout
	}
}

// run delivers queued records until the queue is closed and drained.
func (s *serviceChatSink) run() {
	defer close(s.done)

	for text := range s.queue {
		s.deliver(text)
	}
}

// deliver sends a single record to the service chat.
func (s *serviceChatSink) deliver(text string) {
	ctx, cancel := context.WithTimeout(context.Background(), serviceChatSendTimeout)
	defer cancel()

	if err := s.sender.SendMessage(ctx, s.chatID, text); err != nil {
		s.errLog.Printf("failed to send record: %v", err)
	}
}

// newServiceChatHandler wraps inner so that ERROR records are also sent to sink.
func newServiceChatHandler(inner slog.Handler, sink *serviceChatSink) *serviceChatHandler {
	return &serviceChatHandler{inner: inner, sink: sink}
}

// Enabled reports whether a record of the given level is handled. ERROR
// records are always handled, even if the configured level is higher.
func (h *serviceChatHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= slog.LevelError || h.inner.Enabled(ctx, level)
}

// Handle writes the record to the wrapped handler and mirrors ERROR records
// to the service chat.
func (h *serviceChatHandler) Handle(ctx context.Context, record slog.Record) error {
	var err error

	if h.inner.Enabled(ctx, record.Level) {
		err = h.inner.Handle(ctx, record)
	}

	if record.Level >= slog.LevelError {
		h.sink.send(h.format(record))
	}

	return err
}

// WithAttrs delegates to the wrapped handler and keeps the attributes for the
// service chat message as well.
func (h *serviceChatHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	clone := h.clone()
	clone.inner = h.inner.WithAttrs(attrs)

	prefix := strings.Join(h.groups, ".")
	for _, attr := range attrs {
		clone.attrs = append(clone.attrs, formatAttr(prefix, attr)...)
	}

	return clone
}

// WithGroup delegates to the wrapped handler and qualifies the keys of the
// service chat message with the group name.
func (h *serviceChatHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	clone := h.clone()
	clone.inner = h.inner.WithGroup(name)
	clone.groups = append(clone.groups, name)

	return clone
}

// clone copies the handler with its own attribute and group slices.
func (h *serviceChatHandler) clone() *serviceChatHandler {
	return &serviceChatHandler{
		inner:  h.inner,
		sink:   h.sink,
		attrs:  slices.Clone(h.attrs),
		groups: slices.Clone(h.groups),
	}
}

// format renders a record as a plain text message for the service chat,
// truncated to a single Telegram message.
func (h *serviceChatHandler) format(record slog.Record) string {
	parts := make([]string, 0, len(h.attrs)+record.NumAttrs()+1)
	parts = append(parts, record.Level.String()+": "+record.Message)
	parts = append(parts, h.attrs...)

	prefix := strings.Join(h.groups, ".")

	record.Attrs(func(attr slog.Attr) bool {
		parts = append(parts, formatAttr(prefix, attr)...)

		return true
	})

	return truncateForTelegram(strings.Join(parts, " "))
}

// formatAttr renders an attribute as "key=value" pairs, flattening groups and
// qualifying keys with the prefix of the enclosing groups.
func formatAttr(prefix string, attr slog.Attr) []string {
	attr.Value = attr.Value.Resolve()

	if attr.Value.Kind() == slog.KindGroup {
		group := attr.Value.Group()
		if attr.Key != "" {
			prefix = joinKey(prefix, attr.Key)
		}

		pairs := make([]string, 0, len(group))
		for _, sub := range group {
			pairs = append(pairs, formatAttr(prefix, sub)...)
		}

		return pairs
	}

	if attr.Equal(slog.Attr{}) {
		return nil
	}

	return []string{joinKey(prefix, attr.Key) + "=" + attr.Value.String()}
}

// joinKey qualifies key with the group prefix.
func joinKey(prefix, key string) string {
	if prefix == "" {
		return key
	}

	return prefix + "." + key
}

// truncateForTelegram cuts text down to a single Telegram message.
func truncateForTelegram(text string) string {
	chunks := models.SplitText(text, models.TelegramMessageLimit)
	if len(chunks) == 0 {
		return ""
	}

	return chunks[0]
}
