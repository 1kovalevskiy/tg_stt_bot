package telegram

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

const testToken = "1234567890:AAF-secret-token-value"

// fakeBotAPI is a hand-written fake of the botAPI interface.
type fakeBotAPI struct {
	getFileParams *bot.GetFileParams
	getFileResult *tgmodels.File
	getFileErr    error

	sendParams []*bot.SendMessageParams
	sendErr    error
}

func (f *fakeBotAPI) GetFile(_ context.Context, params *bot.GetFileParams) (*tgmodels.File, error) {
	f.getFileParams = params
	if f.getFileErr != nil {
		return nil, f.getFileErr
	}

	return f.getFileResult, nil
}

func (f *fakeBotAPI) SendMessage(_ context.Context, params *bot.SendMessageParams) (*tgmodels.Message, error) {
	f.sendParams = append(f.sendParams, params)
	if f.sendErr != nil {
		return nil, f.sendErr
	}

	return &tgmodels.Message{}, nil
}

// fakeDoer is a hand-written fake of the httpDoer interface.
type fakeDoer struct {
	gotURL string
	resp   *http.Response
	err    error
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.gotURL = req.URL.String()
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

func TestDownloadFile_Success(t *testing.T) {
	api := &fakeBotAPI{
		getFileResult: &tgmodels.File{FileID: "file-1", FilePath: "voice/file_1.oga"},
	}
	body := &closeTrackingBody{Reader: strings.NewReader("audio-bytes")}
	doer := &fakeDoer{resp: &http.Response{StatusCode: http.StatusOK, Body: body}}

	provider := NewProvider(api, testToken, doer)

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

func TestDownloadFile_GetFileError(t *testing.T) {
	// The library error simulates a transport failure that embeds the
	// request URL — including the token.
	api := &fakeBotAPI{
		getFileErr: errors.New(`Get "https://api.telegram.org/bot` + testToken + `/getFile": connection refused`),
	}
	provider := NewProvider(api, testToken, &fakeDoer{})

	_, err := provider.DownloadFile(context.Background(), "file-1")
	if !errors.Is(err, providers.ErrTelegramGetFile) {
		t.Fatalf("DownloadFile() error = %v, want errors.Is providers.ErrTelegramGetFile", err)
	}

	if strings.Contains(err.Error(), testToken) {
		t.Errorf("DownloadFile() error %q contains the bot token", err.Error())
	}

	if !strings.Contains(err.Error(), redactedToken) {
		t.Errorf("DownloadFile() error %q does not contain the redaction marker", err.Error())
	}
}

func TestDownloadFile_EmptyFilePath(t *testing.T) {
	api := &fakeBotAPI{getFileResult: &tgmodels.File{FileID: "file-1"}}
	provider := NewProvider(api, testToken, &fakeDoer{})

	_, err := provider.DownloadFile(context.Background(), "file-1")
	if !errors.Is(err, providers.ErrTelegramEmptyFilePath) {
		t.Fatalf("DownloadFile() error = %v, want errors.Is providers.ErrTelegramEmptyFilePath", err)
	}
}

func TestDownloadFile_HTTPError(t *testing.T) {
	api := &fakeBotAPI{
		getFileResult: &tgmodels.File{FileID: "file-1", FilePath: "voice/file_1.oga"},
	}
	doer := &fakeDoer{
		err: errors.New(`Get "https://api.telegram.org/file/bot` + testToken + `/voice/file_1.oga": connection refused`),
	}
	provider := NewProvider(api, testToken, doer)

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
	provider := NewProvider(api, testToken, doer)

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
	provider := NewProvider(api, testToken, &fakeDoer{})

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

func TestSendMessage_Error(t *testing.T) {
	api := &fakeBotAPI{
		sendErr: errors.New(`Post "https://api.telegram.org/bot` + testToken + `/sendMessage": timeout`),
	}
	provider := NewProvider(api, testToken, &fakeDoer{})

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
	provider := NewProvider(api, testToken, &fakeDoer{})

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

func TestSendReply_Error(t *testing.T) {
	api := &fakeBotAPI{sendErr: errors.New("chat not found")}
	provider := NewProvider(api, testToken, &fakeDoer{})

	err := provider.SendReply(context.Background(), 42, 777, "transcript")
	if !errors.Is(err, providers.ErrTelegramSendMessage) {
		t.Fatalf("SendReply() error = %v, want errors.Is providers.ErrTelegramSendMessage", err)
	}

	if !strings.Contains(err.Error(), "chat not found") {
		t.Errorf("SendReply() error = %q, want it to contain the underlying message", err.Error())
	}
}
