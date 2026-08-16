package telegram

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/1kovalevskiy/tg_stt_bot/internal/transport"
)

// discardHandler is the base log handler the client tests run against: the
// library reports its own failures through it, and the tests do not read them.
func discardHandler() slog.Handler {
	return slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})
}

// TestNewBotClient_RejectsAnEmptyToken covers the only failure the library
// reports without a network call. The successful path talks to the Bot API
// (getMe on startup) and is therefore not exercised here.
func TestNewBotClient_RejectsAnEmptyToken(t *testing.T) {
	t.Parallel()

	client, err := NewBotClient(fakeConfig{token: ""}, discardHandler())
	if !errors.Is(err, transport.ErrTelegramCreateBotClient) {
		t.Fatalf("NewBotClient() error = %v, want ErrTelegramCreateBotClient", err)
	}

	if client != nil {
		t.Error("NewBotClient() returned a client together with an error")
	}
}

// TestNewBotClient_ErrorDoesNotLeakTheToken pins the redaction: the wrapped
// text is logged and mirrored to the service chat, and the library embeds the
// request URL — token included — into its own errors.
func TestNewBotClient_ErrorDoesNotLeakTheToken(t *testing.T) {
	t.Parallel()

	// A token of blanks is what the library trims down to "empty", so the
	// failure happens offline while the config still reports a token.
	const blankToken = "\t\n"

	_, err := NewBotClient(fakeConfig{token: blankToken}, discardHandler())
	if err == nil {
		t.Fatal("NewBotClient() error = nil, want a failure for a blank token")
	}

	if strings.Contains(err.Error(), blankToken) {
		t.Errorf("NewBotClient() error = %q, want the token replaced with %q", err, models.RedactedToken)
	}
}

// TestIgnoreUpdate_LogsNothing pins the default handler: the library's own
// default prints every unmatched update and leaks message content. It captures
// the global logger, so it cannot run in parallel.
func TestIgnoreUpdate_LogsNothing(t *testing.T) {
	logOut := captureDefaultLogger(t)

	ignoreUpdate(context.Background(), nil, textUpdate(testForeignID, "секретный текст"))

	if logOut.Len() != 0 {
		t.Errorf("log = %q, want an unmatched update dropped silently", logOut.String())
	}
}
