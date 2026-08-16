package telegram

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

const (
	testToken = "1234567890:AAF-secret-token-value"
	// testAPITimeout and testDownloadTimeout are generous enough not to fire
	// during a test, and different from each other so tests can tell which of
	// the two a call used.
	testAPITimeout      = 30 * time.Second
	testDownloadTimeout = 2 * time.Minute
)

// fakeConfig is a hand-written fake of the configProvider interface.
type fakeConfig struct {
	token           string
	apiTimeout      time.Duration
	downloadTimeout time.Duration
}

func (c fakeConfig) GetTelegramToken() string {
	return c.token
}

func (c fakeConfig) GetTelegramAPITimeout() time.Duration {
	return c.apiTimeout
}

func (c fakeConfig) GetTelegramDownloadTimeout() time.Duration {
	return c.downloadTimeout
}

// newTestProvider builds a Provider over a fake config with the test timeouts.
func newTestProvider(api botAPI, client httpDoer) *Provider {
	return NewProvider(api, fakeConfig{
		token:           testToken,
		apiTimeout:      testAPITimeout,
		downloadTimeout: testDownloadTimeout,
	}, client)
}

// fakeBotAPI is a hand-written fake of the botAPI interface.
type fakeBotAPI struct {
	getFileCtx    context.Context
	getFileParams *bot.GetFileParams
	getFileResult *tgmodels.File
	getFileErr    error

	sendCtx    []context.Context
	sendParams []*bot.SendMessageParams
	sendErr    error
}

func (f *fakeBotAPI) GetFile(ctx context.Context, params *bot.GetFileParams) (*tgmodels.File, error) {
	f.getFileCtx = ctx
	f.getFileParams = params

	if f.getFileErr != nil {
		return nil, f.getFileErr
	}

	return f.getFileResult, nil
}

func (f *fakeBotAPI) SendMessage(ctx context.Context, params *bot.SendMessageParams) (*tgmodels.Message, error) {
	f.sendCtx = append(f.sendCtx, ctx)
	f.sendParams = append(f.sendParams, params)

	if f.sendErr != nil {
		return nil, f.sendErr
	}

	return &tgmodels.Message{}, nil
}

// fakeDoer is a hand-written fake of the httpDoer interface.
type fakeDoer struct {
	gotURL string
	gotCtx context.Context
	resp   *http.Response
	err    error
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.gotURL = req.URL.String()
	f.gotCtx = req.Context()

	if f.err != nil {
		return nil, f.err
	}

	return f.resp, nil
}

// closeTrackingBody records whether Close was called.
type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true

	return nil
}

// remainingTimeout reports how much of a context's deadline is left.
func remainingTimeout(t *testing.T, ctx context.Context) time.Duration {
	t.Helper()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("context has no deadline, want the configured timeout applied")
	}

	return time.Until(deadline)
}

func TestDownloadFile_Success(t *testing.T) {
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

func TestDownloadFile_GetFileUsesAPITimeout(t *testing.T) {
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

func TestSendMessage_Success(t *testing.T) {
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

func TestSendReply_Success(t *testing.T) {
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

func TestWrapRedacted_EmptyTokenLeavesMessageIntact(t *testing.T) {
	provider := NewProvider(&fakeBotAPI{}, fakeConfig{apiTimeout: testAPITimeout}, &fakeDoer{})

	err := provider.wrapRedacted(providers.ErrTelegramSendMessage, errors.New("chat not found"))
	if !strings.Contains(err.Error(), "chat not found") {
		t.Errorf("wrapRedacted() = %q, want it to contain the underlying message", err.Error())
	}

	if strings.Contains(err.Error(), models.RedactedToken) {
		t.Errorf("wrapRedacted() = %q, want no redaction marker for an empty token", err.Error())
	}
}
