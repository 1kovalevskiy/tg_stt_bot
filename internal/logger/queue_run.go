package logger

import (
	"context"
	"fmt"
	"time"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

// runQueue delivers queued records until the queue is closed and drained,
// dropping everything above the rate limit.
func (s *ServiceChatSink) runQueue() {
	defer close(s.done)

	for text := range s.queue {
		s.rotateWindow(time.Now())

		if s.windowSent >= models.ServiceChatRateBurst {
			s.suppressed++

			continue
		}

		s.windowSent++
		s.deliverRecord(text)
	}

	s.reportSuppressedRecords()
}

// rotateWindow starts a new rate window once the current one has expired,
// reporting what the expired one had to drop.
func (s *ServiceChatSink) rotateWindow(now time.Time) {
	if !s.windowStart.IsZero() && now.Sub(s.windowStart) < models.ServiceChatRateWindow {
		return
	}

	s.reportSuppressedRecords()

	s.windowStart = now
	s.windowSent = 0
}

// reportSuppressedRecords sends a summary of the records the rate limit
// dropped. The summary itself bypasses the limit: it is at most one message per
// window and is the only trace those records leave in the chat.
func (s *ServiceChatSink) reportSuppressedRecords() {
	if s.suppressed == 0 {
		return
	}

	count := s.suppressed
	s.suppressed = 0

	s.deliverRecord(fmt.Sprintf(models.MsgSuppressedFormat, count))
}

// deliverRecord sends a single record to the service chat. The context is
// created here because the logging caller has none to pass down.
func (s *ServiceChatSink) deliverRecord(text string) {
	ctx, cancel := context.WithTimeout(context.Background(), models.ServiceChatSendTimeout)
	defer cancel()

	if err := s.sender.SendMessage(ctx, s.config.GetTelegramServiceChatID(), text); err != nil {
		s.errLog.Printf("failed to send record: %v", err)
	}
}
