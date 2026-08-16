package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
)

func TestSendReply_Success(t *testing.T) {
	t.Parallel()

	api := &fakeBotAPI{}
	provider := newTestProvider(api, &fakeDoer{})

	if err := provider.SendReply(context.Background(), 42, 777, "transcript"); err != nil {
		t.Fatalf("SendReply() unexpected error: %v", err)
	}

	if len(api.sendParams) != 1 {
		t.Fatalf("SendMessage called %d times, want 1", len(api.sendParams))
	}

	params := api.sendParams[0]
	if params.ChatID != int64(42) {
		t.Errorf("ChatID = %v, want int64(42)", params.ChatID)
	}

	if params.Text != "transcript" {
		t.Errorf("Text = %q, want %q", params.Text, "transcript")
	}

	if params.ReplyParameters == nil {
		t.Fatal("ReplyParameters = nil, want reply parameters set")
	}

	if params.ReplyParameters.MessageID != 777 {
		t.Errorf("ReplyParameters.MessageID = %d, want 777", params.ReplyParameters.MessageID)
	}

	if !params.ReplyParameters.AllowSendingWithoutReply {
		t.Error("ReplyParameters.AllowSendingWithoutReply = false, want true")
	}
}

func TestSendReply_AppliesAPITimeout(t *testing.T) {
	t.Parallel()

	api := &fakeBotAPI{}
	provider := newTestProvider(api, &fakeDoer{})

	if err := provider.SendReply(context.Background(), 42, 777, "transcript"); err != nil {
		t.Fatalf("SendReply() unexpected error: %v", err)
	}

	if len(api.sendCtx) != 1 {
		t.Fatalf("SendMessage called %d times, want 1", len(api.sendCtx))
	}

	if left := remainingTimeout(t, api.sendCtx[0]); left > testAPITimeout {
		t.Errorf("SendReply context has %v left, want at most the API timeout %v", left, testAPITimeout)
	}
}

func TestSendReply_Error(t *testing.T) {
	t.Parallel()

	api := &fakeBotAPI{sendErr: errors.New("chat not found")}
	provider := newTestProvider(api, &fakeDoer{})

	err := provider.SendReply(context.Background(), 42, 777, "transcript")
	if !errors.Is(err, providers.ErrTelegramSendMessage) {
		t.Fatalf("SendReply() error = %v, want errors.Is providers.ErrTelegramSendMessage", err)
	}

	if !strings.Contains(err.Error(), "chat not found") {
		t.Errorf("SendReply() error = %q, want it to contain the underlying message", err.Error())
	}
}

// TestSendReply_CanceledContext pins the shutdown path of the last stage of a
// transcription: the reply is the call most likely to be in flight when the
// bot stops, and its redacted error must still match context.Canceled.
func TestSendReply_CanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider := newTestProvider(&fakeBotAPI{}, &fakeDoer{})

	err := provider.SendReply(ctx, 42, 777, "transcript")
	assertCanceledError(t, err, providers.ErrTelegramSendMessage)
}
