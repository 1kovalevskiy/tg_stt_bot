package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
)

func TestSendMessage_Success(t *testing.T) {
	t.Parallel()

	api := &fakeBotAPI{}
	provider := newTestProvider(api, &fakeDoer{})

	if err := provider.SendMessage(context.Background(), 42, "hello"); err != nil {
		t.Fatalf("SendMessage() unexpected error: %v", err)
	}

	if len(api.sendParams) != 1 {
		t.Fatalf("SendMessage called %d times, want 1", len(api.sendParams))
	}

	params := api.sendParams[0]
	if params.ChatID != int64(42) {
		t.Errorf("ChatID = %v, want int64(42)", params.ChatID)
	}

	if params.Text != "hello" {
		t.Errorf("Text = %q, want %q", params.Text, "hello")
	}

	if params.ReplyParameters != nil {
		t.Errorf("ReplyParameters = %+v, want nil for plain message", params.ReplyParameters)
	}
}

func TestSendMessage_AppliesAPITimeout(t *testing.T) {
	t.Parallel()

	api := &fakeBotAPI{}
	provider := newTestProvider(api, &fakeDoer{})

	if err := provider.SendMessage(context.Background(), 42, "hello"); err != nil {
		t.Fatalf("SendMessage() unexpected error: %v", err)
	}

	if len(api.sendCtx) != 1 {
		t.Fatalf("SendMessage called %d times, want 1", len(api.sendCtx))
	}

	if left := remainingTimeout(t, api.sendCtx[0]); left > testAPITimeout {
		t.Errorf("SendMessage context has %v left, want at most the API timeout %v", left, testAPITimeout)
	}

	// The context belongs to the call and is released when it returns.
	if !errors.Is(api.sendCtx[0].Err(), context.Canceled) {
		t.Errorf("SendMessage context error = %v, want context.Canceled", api.sendCtx[0].Err())
	}
}

func TestSendMessage_Error(t *testing.T) {
	t.Parallel()

	api := &fakeBotAPI{
		sendErr: errors.New(`Post "https://api.telegram.org/bot` + testToken + `/sendMessage": timeout`),
	}
	provider := newTestProvider(api, &fakeDoer{})

	err := provider.SendMessage(context.Background(), 42, "hello")
	if !errors.Is(err, providers.ErrTelegramSendMessage) {
		t.Fatalf("SendMessage() error = %v, want errors.Is providers.ErrTelegramSendMessage", err)
	}

	if strings.Contains(err.Error(), testToken) {
		t.Errorf("SendMessage() error %q contains the bot token", err.Error())
	}
}

// TestSendMessage_CanceledContext pins the shutdown path: the redacted error
// must still match context.Canceled, or the transport reports the stopping bot
// as a failure and mirrors it into the service chat being drained.
func TestSendMessage_CanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider := newTestProvider(&fakeBotAPI{}, &fakeDoer{})

	err := provider.SendMessage(ctx, 42, "hello")
	assertCanceledError(t, err, providers.ErrTelegramSendMessage)
}
