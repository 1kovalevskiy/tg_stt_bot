package logger

import (
	"strings"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

// Write makes the sink an io.Writer, so a standard slog handler can render the
// records mirrored to the service chat. It never fails: a delivery problem is
// reported to the sink's own error log, not back to the logging caller.
func (s *ServiceChatSink) Write(record []byte) (int, error) {
	text := strings.TrimRight(string(record), "\n")

	s.sendRecord(models.TruncateText(text, models.TelegramMessageLimit))

	return len(record), nil
}

// sendRecord queues text for delivery without ever blocking the caller: an
// overflow or a closed sink drops the record and reports it to stderr.
func (s *ServiceChatSink) sendRecord(text string) {
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
