package telegram

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
	tgmodels "github.com/go-telegram/bot/models"
)

func TestDownloadFile_Success(t *testing.T) {
	t.Parallel()

	api := &fakeBotAPI{
		getFileResult: &tgmodels.File{FileID: "file-1", FilePath: "voice/file_1.oga"},
	}
	body := &closeTrackingBody{Reader: strings.NewReader("audio-bytes")}
	doer := &fakeDoer{resp: &http.Response{StatusCode: http.StatusOK, Body: body}}

	provider := newTestProvider(api, doer)

	reader, err := provider.DownloadFile(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("DownloadFile() unexpected error: %v", err)
	}

	if api.getFileParams == nil || api.getFileParams.FileID != "file-1" {
		t.Errorf("GetFile params = %+v, want FileID %q", api.getFileParams, "file-1")
	}

	wantURL := "https://api.telegram.org/file/bot" + testToken + "/voice/file_1.oga"
	if doer.gotURL != wantURL {
		t.Errorf("download URL = %q, want %q", doer.gotURL, wantURL)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read downloaded body: %v", err)
	}

	if string(content) != "audio-bytes" {
		t.Errorf("downloaded content = %q, want %q", content, "audio-bytes")
	}

	if err := reader.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	if !body.closed {
		t.Error("response body not closed after Close()")
	}
}

// TestDownloadFile_TimeoutCoversBodyRead pins the exception to the usual
// defer cancel(): the caller reads the stream after DownloadFile returns, so
// the download context must stay alive until the caller closes the reader.
func TestDownloadFile_TimeoutCoversBodyRead(t *testing.T) {
	t.Parallel()

	api := &fakeBotAPI{
		getFileResult: &tgmodels.File{FileID: "file-1", FilePath: "voice/file_1.oga"},
	}
	body := &closeTrackingBody{Reader: strings.NewReader("audio-bytes")}
	doer := &fakeDoer{resp: &http.Response{StatusCode: http.StatusOK, Body: body}}

	provider := newTestProvider(api, doer)

	reader, err := provider.DownloadFile(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("DownloadFile() unexpected error: %v", err)
	}

	if err := doer.gotCtx.Err(); err != nil {
		t.Fatalf("download context canceled right after DownloadFile() returned: %v", err)
	}

	if left := remainingTimeout(t, doer.gotCtx); left <= testAPITimeout {
		t.Errorf("download context has %v left, want more than the API timeout %v", left, testAPITimeout)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read downloaded body after DownloadFile() returned: %v", err)
	}

	if string(content) != "audio-bytes" {
		t.Errorf("downloaded content = %q, want %q", content, "audio-bytes")
	}

	if err := reader.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	if !errors.Is(doer.gotCtx.Err(), context.Canceled) {
		t.Errorf("download context error after Close() = %v, want context.Canceled", doer.gotCtx.Err())
	}

	// Closing twice must stay safe: cancel is idempotent.
	if err := reader.Close(); err != nil {
		t.Fatalf("second Close() unexpected error: %v", err)
	}
}

// TestDownloadFile_ReleasesContextOnError guards against a context leak on the
// paths where no reader is handed to the caller and nothing can call cancel.
func TestDownloadFile_ReleasesContextOnError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		doer *fakeDoer
	}{
		{
			name: "transport error",
			doer: &fakeDoer{err: errors.New("connection refused")},
		},
		{
			name: "unexpected status",
			doer: &fakeDoer{resp: &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       &closeTrackingBody{Reader: strings.NewReader(`{"ok":false}`)},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := &fakeBotAPI{
				getFileResult: &tgmodels.File{FileID: "file-1", FilePath: "voice/file_1.oga"},
			}
			provider := newTestProvider(api, tt.doer)

			if _, err := provider.DownloadFile(context.Background(), "file-1"); err == nil {
				t.Fatal("DownloadFile() expected error, got nil")
			}

			if !errors.Is(tt.doer.gotCtx.Err(), context.Canceled) {
				t.Errorf("download context error = %v, want context.Canceled", tt.doer.gotCtx.Err())
			}
		})
	}
}

func TestDownloadFile_GetFileError(t *testing.T) {
	t.Parallel()

	// The library error simulates a transport failure that embeds the
	// request URL — including the token.
	api := &fakeBotAPI{
		getFileErr: errors.New(`Get "https://api.telegram.org/bot` + testToken + `/getFile": connection refused`),
	}
	provider := newTestProvider(api, &fakeDoer{})

	_, err := provider.DownloadFile(context.Background(), "file-1")
	if !errors.Is(err, providers.ErrTelegramGetFile) {
		t.Fatalf("DownloadFile() error = %v, want errors.Is providers.ErrTelegramGetFile", err)
	}

	if strings.Contains(err.Error(), testToken) {
		t.Errorf("DownloadFile() error %q contains the bot token", err.Error())
	}

	if !strings.Contains(err.Error(), models.RedactedToken) {
		t.Errorf("DownloadFile() error %q does not contain the redaction marker", err.Error())
	}

	if !errors.Is(api.getFileCtx.Err(), context.Canceled) {
		t.Errorf("getFile context error = %v, want context.Canceled", api.getFileCtx.Err())
	}
}

// TestDownloadFile_CanceledContext pins the shutdown path of both stages of
// the download. The redacted error still has to match context.Canceled: the
// transport logs a controller failure at ERROR and mirrors it into the service
// chat, which is being drained exactly when the shutdown cancels the download.
func TestDownloadFile_CanceledContext(t *testing.T) {
	t.Parallel()

	t.Run("get file", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		api := &fakeBotAPI{
			getFileResult: &tgmodels.File{FileID: "file-1", FilePath: "voice/file_1.oga"},
		}
		provider := newTestProvider(api, &fakeDoer{})

		_, err := provider.DownloadFile(ctx, "file-1")
		assertCanceledError(t, err, providers.ErrTelegramGetFile)
	})

	t.Run("body download", func(t *testing.T) {
		t.Parallel()

		// The bot stops right after the metadata call: the download itself is
		// the in-flight request the shutdown cuts short.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		api := &fakeBotAPI{
			getFileResult: &tgmodels.File{FileID: "file-1", FilePath: "voice/file_1.oga"},
			afterGetFile:  cancel,
		}
		doer := &fakeDoer{}
		provider := newTestProvider(api, doer)

		_, err := provider.DownloadFile(ctx, "file-1")
		assertCanceledError(t, err, providers.ErrTelegramDownloadFailed)

		if !strings.Contains(doer.gotURL, testToken) {
			t.Fatalf("download URL = %q, want the token in the URL the error quotes", doer.gotURL)
		}
	})
}

func TestDownloadFile_GetFileUsesAPITimeout(t *testing.T) {
	t.Parallel()

	api := &fakeBotAPI{
		getFileResult: &tgmodels.File{FileID: "file-1", FilePath: "voice/file_1.oga"},
	}
	body := &closeTrackingBody{Reader: strings.NewReader("audio-bytes")}
	doer := &fakeDoer{resp: &http.Response{StatusCode: http.StatusOK, Body: body}}

	provider := newTestProvider(api, doer)

	reader, err := provider.DownloadFile(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("DownloadFile() unexpected error: %v", err)
	}
	defer reader.Close()

	if left := remainingTimeout(t, api.getFileCtx); left > testAPITimeout {
		t.Errorf("getFile context has %v left, want at most the API timeout %v", left, testAPITimeout)
	}
}

func TestDownloadFile_NoUsableFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file *tgmodels.File
	}{
		{name: "empty file path", file: &tgmodels.File{FileID: "file-1"}},
		// getFile answering with neither a file nor an error is not something
		// the Bot API promises, but the library types it as possible and a
		// dereference here would panic the worker and kill the process.
		{name: "nil file", file: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api := &fakeBotAPI{getFileResult: tt.file}
			doer := &fakeDoer{}
			provider := newTestProvider(api, doer)

			_, err := provider.DownloadFile(context.Background(), "file-1")
			if !errors.Is(err, providers.ErrTelegramEmptyFilePath) {
				t.Fatalf("DownloadFile() error = %v, want errors.Is providers.ErrTelegramEmptyFilePath", err)
			}

			if doer.gotURL != "" {
				t.Errorf("download requested %q, want no download attempt", doer.gotURL)
			}
		})
	}
}

func TestDownloadFile_HTTPError(t *testing.T) {
	t.Parallel()

	api := &fakeBotAPI{
		getFileResult: &tgmodels.File{FileID: "file-1", FilePath: "voice/file_1.oga"},
	}
	doer := &fakeDoer{
		err: errors.New(`Get "https://api.telegram.org/file/bot` + testToken + `/voice/file_1.oga": connection refused`),
	}
	provider := newTestProvider(api, doer)

	_, err := provider.DownloadFile(context.Background(), "file-1")
	if !errors.Is(err, providers.ErrTelegramDownloadFailed) {
		t.Fatalf("DownloadFile() error = %v, want errors.Is providers.ErrTelegramDownloadFailed", err)
	}

	if strings.Contains(err.Error(), testToken) {
		t.Errorf("DownloadFile() error %q contains the bot token", err.Error())
	}
}

func TestDownloadFile_UnexpectedStatus(t *testing.T) {
	t.Parallel()

	api := &fakeBotAPI{
		getFileResult: &tgmodels.File{FileID: "file-1", FilePath: "voice/file_1.oga"},
	}
	body := &closeTrackingBody{Reader: strings.NewReader(`{"ok":false}`)}
	doer := &fakeDoer{resp: &http.Response{StatusCode: http.StatusNotFound, Body: body}}
	provider := newTestProvider(api, doer)

	_, err := provider.DownloadFile(context.Background(), "file-1")
	if !errors.Is(err, providers.ErrTelegramUnexpectedStatus) {
		t.Fatalf("DownloadFile() error = %v, want errors.Is providers.ErrTelegramUnexpectedStatus", err)
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("DownloadFile() error = %q, want it to contain status code", err.Error())
	}

	if !body.closed {
		t.Error("response body not closed on non-200 status")
	}
}
