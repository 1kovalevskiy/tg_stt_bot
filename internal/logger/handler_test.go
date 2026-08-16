package logger

import (
	"bytes"
	"context"
	"io"
	"log"
	"log/slog"
	"strings"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

func TestServiceChatHandler_ErrorGoesToServiceChat(t *testing.T) {
	sender := &fakeSender{}
	logger, stdout, errOut, sink := newTestLogger(t, sender, models.ServiceChatQueueSize)

	logger.Error("stt is down", "err", "connection refused")

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	sent := sender.sent()
	if len(sent) != 1 {
		t.Fatalf("sender got %d messages, want 1", len(sent))
	}

	if !strings.Contains(sent[0], "stt is down") || !strings.Contains(sent[0], `err="connection refused"`) {
		t.Errorf("service chat message = %q, want the message and its attrs", sent[0])
	}

	if !strings.Contains(sent[0], "level="+slog.LevelError.String()) {
		t.Errorf("service chat message = %q, want it to carry the level", sent[0])
	}

	// Telegram stamps every message itself, so the record timestamp is dropped.
	if strings.Contains(sent[0], "time=") {
		t.Errorf("service chat message = %q, want the timestamp dropped", sent[0])
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
	logger, stdout, _, sink := newTestLogger(t, sender, models.ServiceChatQueueSize)

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
	sink := newServiceChatSink(sender, testConfig(), models.ServiceChatQueueSize, log.New(io.Discard, "", 0))
	logger := slog.New(NewServiceChatHandler(inner, sink))

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
	logger, _, _, sink := newTestLogger(t, sender, models.ServiceChatQueueSize)

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
	logger, _, _, sink := newTestLogger(t, sender, models.ServiceChatQueueSize)

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
	logger, _, _, sink := newTestLogger(t, sender, models.ServiceChatQueueSize)

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

func TestNewBaseHandler_TakesTheLevelFromTheConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		level       string
		record      slog.Level
		wantEnabled bool
	}{
		{name: "debug config takes debug", level: "DEBUG", record: slog.LevelDebug, wantEnabled: true},
		{name: "info config drops debug", level: "INFO", record: slog.LevelDebug, wantEnabled: false},
		{name: "info config takes error", level: "INFO", record: slog.LevelError, wantEnabled: true},
		{name: "error config drops warn", level: "ERROR", record: slog.LevelWarn, wantEnabled: false},
		{name: "unknown config behaves as info", level: "verbose", record: slog.LevelInfo, wantEnabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := NewBaseHandler(fakeConfig{level: tt.level})

			if got := handler.Enabled(context.Background(), tt.record); got != tt.wantEnabled {
				t.Errorf("Enabled(%v) = %v for level %q, want %v", tt.record, got, tt.level, tt.wantEnabled)
			}
		})
	}
}
