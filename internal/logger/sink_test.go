package logger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

const (
	serviceChatID int64 = -100777
	testLogLevel        = "INFO"
)

// fakeSender is a hand-written fake of the messageSender interface.
type fakeSender struct {
	mu       sync.Mutex
	messages []string
	chatIDs  []int64

	err error
	// block, when non-nil, holds every send until it is closed.
	block chan struct{}
}

func (f *fakeSender) SendMessage(_ context.Context, chatID int64, text string) error {
	if f.block != nil {
		<-f.block
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.chatIDs = append(f.chatIDs, chatID)
	f.messages = append(f.messages, text)

	return f.err
}

func (f *fakeSender) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.messages...)
}

func (f *fakeSender) sentChatIDs() []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]int64(nil), f.chatIDs...)
}

// fakeConfig is a hand-written fake of the config interfaces this package
// consumes: the service chat id and the log level.
type fakeConfig struct {
	chatID int64
	level  string
}

func (f fakeConfig) GetTelegramServiceChatID() int64 { return f.chatID }

func (f fakeConfig) GetAppLogLevel() string { return f.level }

func testConfig() fakeConfig {
	return fakeConfig{chatID: serviceChatID, level: testLogLevel}
}

// newTestLogger builds a logger with the fan-out handler, returning the
// stdout buffer, the sink's own error log buffer and the sink.
func newTestLogger(
	t *testing.T, sender *fakeSender, queueSize int,
) (*slog.Logger, *bytes.Buffer, *bytes.Buffer, *ServiceChatSink) {
	t.Helper()

	stdout := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	inner := slog.NewTextHandler(stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	sink := newServiceChatSink(sender, testConfig(), queueSize, log.New(errOut, "", 0))

	return slog.New(NewServiceChatHandler(inner, sink)), stdout, errOut, sink
}

func TestNewServiceChatSink_DeliversToTheConfiguredChat(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	sink := NewServiceChatSink(sender, testConfig())

	if _, err := sink.Write([]byte("level=ERROR msg=boom\n")); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	if got := cap(sink.queue); got != models.ServiceChatQueueSize {
		t.Errorf("queue capacity = %d, want models.ServiceChatQueueSize %d", got, models.ServiceChatQueueSize)
	}

	if ids := sender.sentChatIDs(); len(ids) != 1 || ids[0] != serviceChatID {
		t.Errorf("sender got chat ids %v, want [%d]", ids, serviceChatID)
	}
}

// TestServiceChatSink_OverflowDoesNotBlock asserts a wall-clock budget on a
// blocked delivery goroutine, so it stays sequential: under a parallel run the
// contention, and not the sink, would decide whether it passes.
func TestServiceChatSink_OverflowDoesNotBlock(t *testing.T) {
	release := make(chan struct{})
	sender := &fakeSender{block: release}
	logger, _, errOut, sink := newTestLogger(t, sender, 1)

	done := make(chan struct{})

	go func() {
		defer close(done)

		// Far more records than the queue holds; the first one is taken by
		// the blocked delivery goroutine, the rest must be dropped.
		for i := 0; i < 100; i++ {
			logger.Error("overflow record")
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("logging blocked on a full service chat queue")
	}

	close(release)

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	if !strings.Contains(errOut.String(), "queue is full") {
		t.Errorf("sink error log = %q, want a drop report", errOut.String())
	}

	// One record is held by the blocked delivery, one more fits the queue of
	// size 1: everything else must have been dropped.
	if got := len(sender.sent()); got > 2 {
		t.Errorf("sender got %d messages, want at most 2 for a queue of size 1", got)
	}
}

func TestServiceChatSink_SendFailureIsReportedWithoutRecursion(t *testing.T) {
	t.Parallel()

	sendErr := errors.New("chat not found")
	sender := &fakeSender{err: sendErr}
	logger, _, errOut, sink := newTestLogger(t, sender, models.ServiceChatQueueSize)

	logger.Error("stt is down")

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	if got := len(sender.sent()); got != 1 {
		t.Fatalf("sender got %d messages, want exactly 1 (a retry loop would send more)", got)
	}

	if !strings.Contains(errOut.String(), sendErr.Error()) {
		t.Errorf("sink error log = %q, want the send failure", errOut.String())
	}
}

func TestServiceChatSink_CloseDrainsQueue(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	logger, _, _, sink := newTestLogger(t, sender, models.ServiceChatQueueSize)

	const records = 10

	for i := 0; i < records; i++ {
		logger.Error("queued record")
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	if got := len(sender.sent()); got != records {
		t.Errorf("sender got %d messages after Close, want %d", got, records)
	}
}

func TestServiceChatSink_CloseIsIdempotentAndDropsLaterRecords(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	logger, _, errOut, sink := newTestLogger(t, sender, models.ServiceChatQueueSize)

	if err := sink.Close(); err != nil {
		t.Fatalf("first Close() unexpected error: %v", err)
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("second Close() unexpected error: %v", err)
	}

	logger.Error("after shutdown")

	if got := len(sender.sent()); got != 0 {
		t.Errorf("sender got %d messages after Close, want 0", got)
	}

	if !strings.Contains(errOut.String(), "sink is closed") {
		t.Errorf("sink error log = %q, want a closed-sink drop report", errOut.String())
	}
}

func TestServiceChatSink_QueueSizeIsClampedToAtLeastOne(t *testing.T) {
	t.Parallel()

	sink := newServiceChatSink(&fakeSender{}, testConfig(), 0, log.New(io.Discard, "", 0))

	defer func() {
		if err := sink.Close(); err != nil {
			t.Errorf("Close() unexpected error: %v", err)
		}
	}()

	if got := cap(sink.queue); got != models.ServiceChatMinQueueSize {
		t.Errorf("queue capacity = %d, want the floor %d for a non-positive queue size",
			got, models.ServiceChatMinQueueSize)
	}
}

// TestServiceChatSink_CloseReportsDrainTimeout drives a millisecond-scale drain
// deadline, so it stays sequential for the same reason as the overflow test.
func TestServiceChatSink_CloseReportsDrainTimeout(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	sender := &fakeSender{block: release}
	logger, _, _, sink := newTestLogger(t, sender, models.ServiceChatQueueSize)

	// The drain must not outlive a stuck delivery forever; shortened here so
	// the test does not wait out a real send timeout.
	sink.drainTimeout = 50 * time.Millisecond

	logger.Error("stuck record")

	if err := sink.Close(); !errors.Is(err, ErrSinkDrainTimeout) {
		t.Fatalf("Close() error = %v, want ErrSinkDrainTimeout", err)
	}
}

func TestServiceChatSink_RateLimitCapsDeliveriesAndReportsTheRest(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	logger, _, _, sink := newTestLogger(t, sender, models.ServiceChatQueueSize)

	const extra = 5

	for i := 0; i < models.ServiceChatRateBurst+extra; i++ {
		logger.Error("repeating failure")
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	sent := sender.sent()

	// The burst plus a single summary of everything the limit dropped.
	if len(sent) != models.ServiceChatRateBurst+1 {
		t.Fatalf("sender got %d messages, want %d", len(sent), models.ServiceChatRateBurst+1)
	}

	summary := sent[len(sent)-1]
	if !strings.Contains(summary, fmt.Sprintf(models.MsgSuppressedFormat, extra)) {
		t.Errorf("last message = %q, want a summary of %d suppressed records", summary, extra)
	}
}

func TestServiceChatSink_WriteTruncatesToOneMessage(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	sink := newServiceChatSink(sender, testConfig(), models.ServiceChatQueueSize, log.New(io.Discard, "", 0))

	long := strings.Repeat("\U0001F600", models.TelegramMessageLimit)

	n, err := sink.Write([]byte(long + "\n"))
	if err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}

	if n != len(long)+1 {
		t.Errorf("Write() = %d, want the whole input reported as written", n)
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	sent := sender.sent()
	if len(sent) != 1 {
		t.Fatalf("sender got %d messages, want 1", len(sent))
	}

	if chunks := models.SplitText(sent[0], models.TelegramMessageLimit); len(chunks) != 1 {
		t.Errorf("service chat message is %d chunks long, want it to fit into one message", len(chunks))
	}
}
