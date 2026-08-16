package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

// fakeSender is a hand-written fake of the serviceChatSender interface.
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

const serviceChatID int64 = -100777

// newTestLogger builds a logger with the fan-out handler, returning the
// stdout buffer, the sink's own error log buffer and the sink.
func newTestLogger(t *testing.T, sender *fakeSender, queueSize int) (*slog.Logger, *bytes.Buffer, *bytes.Buffer, *serviceChatSink) {
	t.Helper()

	stdout := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	inner := slog.NewTextHandler(stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	sink := newServiceChatSink(sender, serviceChatID, queueSize, log.New(errOut, "", 0))

	return slog.New(newServiceChatHandler(inner, sink)), stdout, errOut, sink
}

func TestServiceChatHandler_ErrorGoesToServiceChat(t *testing.T) {
	sender := &fakeSender{}
	logger, stdout, errOut, sink := newTestLogger(t, sender, serviceChatQueueSize)

	logger.Error("stt is down", "err", "connection refused")

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	sent := sender.sent()
	if len(sent) != 1 {
		t.Fatalf("sender got %d messages, want 1", len(sent))
	}

	if !strings.Contains(sent[0], "stt is down") || !strings.Contains(sent[0], "err=connection refused") {
		t.Errorf("service chat message = %q, want the message and its attrs", sent[0])
	}

	if !strings.HasPrefix(sent[0], slog.LevelError.String()) {
		t.Errorf("service chat message = %q, want it to start with the level", sent[0])
	}

	if ids := sender.sentChatIDs(); len(ids) != 1 || ids[0] != serviceChatID {
		t.Errorf("sender got chat ids %v, want [%d]", ids, serviceChatID)
	}

	if !strings.Contains(stdout.String(), "stt is down") {
		t.Errorf("stdout = %q, want the record as well", stdout.String())
	}

	if errOut.Len() != 0 {
		t.Errorf("sink error log = %q, want it empty", errOut.String())
	}
}

func TestServiceChatHandler_NonErrorStaysInStdout(t *testing.T) {
	sender := &fakeSender{}
	logger, stdout, _, sink := newTestLogger(t, sender, serviceChatQueueSize)

	logger.Info("bot started")
	logger.Warn("slow transcription")
	logger.Debug("not enabled at all")

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	if sent := sender.sent(); len(sent) != 0 {
		t.Errorf("sender got %v, want nothing below ERROR", sent)
	}

	out := stdout.String()
	if !strings.Contains(out, "bot started") || !strings.Contains(out, "slow transcription") {
		t.Errorf("stdout = %q, want the info and warn records", out)
	}

	if strings.Contains(out, "not enabled at all") {
		t.Errorf("stdout = %q, want the debug record filtered out by the inner handler", out)
	}
}

func TestServiceChatHandler_ErrorPassesHigherInnerLevel(t *testing.T) {
	sender := &fakeSender{}
	stdout := &bytes.Buffer{}
	// The inner handler drops everything: ERROR must still reach the sink.
	inner := slog.NewTextHandler(stdout, &slog.HandlerOptions{Level: slog.LevelError + 1})
	sink := newServiceChatSink(sender, serviceChatID, serviceChatQueueSize, log.New(io.Discard, "", 0))
	logger := slog.New(newServiceChatHandler(inner, sink))

	logger.Error("boom")

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	if sent := sender.sent(); len(sent) != 1 {
		t.Fatalf("sender got %d messages, want 1", len(sent))
	}

	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want it empty", stdout.String())
	}
}

func TestServiceChatHandler_WithAttrsAndGroupKeepAttributes(t *testing.T) {
	sender := &fakeSender{}
	logger, _, _, sink := newTestLogger(t, sender, serviceChatQueueSize)

	logger.With("component", "chat").
		WithGroup("audio").
		With("file_id", "abc").
		Error("transcription failed", "chat_id", -100123)

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	sent := sender.sent()
	if len(sent) != 1 {
		t.Fatalf("sender got %d messages, want 1", len(sent))
	}

	for _, want := range []string{
		"transcription failed",
		"component=chat",
		"audio.file_id=abc",
		"audio.chat_id=-100123",
	} {
		if !strings.Contains(sent[0], want) {
			t.Errorf("service chat message = %q, want it to contain %q", sent[0], want)
		}
	}
}

func TestServiceChatHandler_GroupAttrIsFlattened(t *testing.T) {
	sender := &fakeSender{}
	logger, _, _, sink := newTestLogger(t, sender, serviceChatQueueSize)

	logger.Error("failed", slog.Group("stt", slog.String("status", "500"), slog.Int("attempt", 2)))

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	sent := sender.sent()
	if len(sent) != 1 {
		t.Fatalf("sender got %d messages, want 1", len(sent))
	}

	if !strings.Contains(sent[0], "stt.status=500") || !strings.Contains(sent[0], "stt.attempt=2") {
		t.Errorf("service chat message = %q, want the flattened group attrs", sent[0])
	}
}

func TestServiceChatHandler_LongRecordTruncatedToOneMessage(t *testing.T) {
	sender := &fakeSender{}
	logger, _, _, sink := newTestLogger(t, sender, serviceChatQueueSize)

	logger.Error(strings.Repeat("очень длинная ошибка ", 500))

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

	if got := len(sender.sent()); got >= 100 {
		t.Errorf("sender got %d messages, want the overflow dropped", got)
	}
}

func TestServiceChatSink_SendFailureIsReportedWithoutRecursion(t *testing.T) {
	sendErr := errors.New("chat not found")
	sender := &fakeSender{err: sendErr}
	logger, _, errOut, sink := newTestLogger(t, sender, serviceChatQueueSize)

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
	sender := &fakeSender{}
	logger, _, _, sink := newTestLogger(t, sender, serviceChatQueueSize)

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
	sender := &fakeSender{}
	logger, _, errOut, sink := newTestLogger(t, sender, serviceChatQueueSize)

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

func TestParseLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  slog.Level
	}{
		{name: "debug", value: "DEBUG", want: slog.LevelDebug},
		{name: "lowercase info", value: "info", want: slog.LevelInfo},
		{name: "warn", value: "WARN", want: slog.LevelWarn},
		{name: "warning", value: " warning ", want: slog.LevelWarn},
		{name: "error", value: "ERROR", want: slog.LevelError},
		{name: "unknown falls back to info", value: "verbose", want: slog.LevelInfo},
		{name: "empty falls back to info", value: "", want: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := parseLogLevel(tt.value); got != tt.want {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
