package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
)

func TestWrapRedacted_EmptyTokenLeavesMessageIntact(t *testing.T) {
	t.Parallel()

	provider := NewProvider(&fakeBotAPI{}, fakeConfig{apiTimeout: testAPITimeout}, &fakeDoer{})

	err := provider.wrapRedactedError(providers.ErrTelegramSendMessage, errors.New("chat not found"))
	if !strings.Contains(err.Error(), "chat not found") {
		t.Errorf("wrapRedactedError() = %q, want it to contain the underlying message", err.Error())
	}

	if strings.Contains(err.Error(), models.RedactedToken) {
		t.Errorf("wrapRedactedError() = %q, want no redaction marker for an empty token", err.Error())
	}
}

// TestWrapRedacted_TokenIsReplaced pins the redaction itself: the library
// embeds the request URL, token included, into its own errors.
func TestWrapRedacted_TokenIsReplaced(t *testing.T) {
	t.Parallel()

	provider := newTestProvider(&fakeBotAPI{}, &fakeDoer{})

	libraryErr := errors.New(`Post "https://api.telegram.org/bot` + testToken + `/sendMessage": timeout`)

	err := provider.wrapRedactedError(providers.ErrTelegramSendMessage, libraryErr)
	if !errors.Is(err, providers.ErrTelegramSendMessage) {
		t.Fatalf("wrapRedactedError() = %v, want errors.Is providers.ErrTelegramSendMessage", err)
	}

	if strings.Contains(err.Error(), testToken) {
		t.Errorf("wrapRedactedError() = %q, want the token redacted", err.Error())
	}

	if !strings.Contains(err.Error(), models.RedactedToken) {
		t.Errorf("wrapRedactedError() = %q, want the redaction marker", err.Error())
	}
}

// TestWrapRedacted_KeepsContextCause pins the other half of the redaction:
// flattening the underlying error into text drops its chain, so a cancellation
// and a deadline are classified first and wrapped into a sentinel carrying the
// context error. Without it a shutdown reads as a service failure one layer up.
func TestWrapRedacted_KeepsContextCause(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cause     error
		sentinel  error
		wantCause error
	}{
		{
			name:      "canceled",
			cause:     context.Canceled,
			sentinel:  providers.ErrTelegramRequestCanceled,
			wantCause: context.Canceled,
		},
		{
			name:      "deadline exceeded",
			cause:     context.DeadlineExceeded,
			sentinel:  providers.ErrTelegramRequestTimeout,
			wantCause: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := newTestProvider(&fakeBotAPI{}, &fakeDoer{})

			err := provider.wrapRedactedError(
				providers.ErrTelegramGetFile,
				newRequestError(testAPIURL+"/getFile", tt.cause),
			)

			if !errors.Is(err, providers.ErrTelegramGetFile) {
				t.Fatalf("wrapRedactedError() = %v, want errors.Is providers.ErrTelegramGetFile", err)
			}

			if !errors.Is(err, tt.sentinel) {
				t.Errorf("wrapRedactedError() = %v, want errors.Is %v", err, tt.sentinel)
			}

			if !errors.Is(err, tt.wantCause) {
				t.Errorf("wrapRedactedError() = %v, want it to wrap %v", err, tt.wantCause)
			}

			if strings.Contains(err.Error(), testToken) {
				t.Errorf("wrapRedactedError() = %q, want the token redacted", err.Error())
			}

			if !strings.Contains(err.Error(), models.RedactedToken) {
				t.Errorf("wrapRedactedError() = %q, want the redaction marker", err.Error())
			}
		})
	}
}

// TestWrapRedacted_PlainFailureIsNotACancellation guards the other direction:
// a service failure must not be reported as a shutdown and silenced.
func TestWrapRedacted_PlainFailureIsNotACancellation(t *testing.T) {
	t.Parallel()

	provider := newTestProvider(&fakeBotAPI{}, &fakeDoer{})

	err := provider.wrapRedactedError(
		providers.ErrTelegramDownloadFailed,
		newRequestError(testAPIURL+"/voice/file_1.oga", errors.New("connection refused")),
	)

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("wrapRedactedError() = %v, want a plain failure to match neither context error", err)
	}

	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("wrapRedactedError() = %q, want it to contain the underlying message", err.Error())
	}
}
