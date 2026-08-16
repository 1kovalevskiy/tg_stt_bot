package telegram

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/1kovalevskiy/tg_stt_bot/internal/controllers"
	telegramProvider "github.com/1kovalevskiy/tg_stt_bot/internal/providers/telegram"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// The tests below capture the global logger and therefore cannot run in
// parallel: slog.Default is process-wide state.

const (
	// testAPIURL is the Bot API endpoint prefix the provider errors quote: it
	// carries the token, which is why those errors are redacted.
	testAPIURL = "https://api.telegram.org/bot" + testBotToken
	// testProviderTimeout is generous enough never to fire during the test.
	testProviderTimeout = time.Minute
)

func TestDispatcher_ControllerErrorsAreSwallowed(t *testing.T) {
	logOut := captureDefaultLogger(t)

	chat := &fakeChatController{err: errors.New("stt is down")}
	admin := &fakeAdminController{err: errors.New("chat not found")}
	d := NewDispatcher(chat, admin, testConfig())

	// A failing controller must not panic the worker: the error is logged
	// and the update is dropped.
	d.handleVoice(context.Background(), nil, voiceUpdate(testAllowedID, 1))
	d.handleVideoNote(context.Background(), nil, videoNoteUpdate(testAllowedID, 2))
	d.handleAdminCommand(context.Background(), nil, textUpdate(testAdminID, "/status"))

	if len(chat.voice) != 1 || len(chat.videoNote) != 1 || len(admin.calls) != 1 {
		t.Errorf("controllers called %+v / %+v / %+v, want one call each",
			chat.voice, chat.videoNote, admin.calls)
	}

	if got := strings.Count(logOut.String(), "level=ERROR"); got != 3 {
		t.Errorf("logged %d ERROR records, want one per failed update: %q", got, logOut.String())
	}
}

// TestDispatcher_FailedCommandLogsTheParsedName pins what a failed command
// leaves in the log: the command word alone, parsed the way the controller
// parses it, and never the message text around it.
func TestDispatcher_FailedCommandLogsTheParsedName(t *testing.T) {
	logOut := captureDefaultLogger(t)

	admin := &fakeAdminController{err: errors.New("chat not found")}
	d := NewDispatcher(&fakeChatController{}, admin, testConfig())

	d.handleAdminCommand(context.Background(), nil, textUpdate(testAdminID, "/STATUS@my_bot секретный аргумент"))

	out := logOut.String()
	if !strings.Contains(out, "command=/status") {
		t.Errorf("log = %q, want the lowercased command name", out)
	}

	if strings.Contains(out, "секретный аргумент") {
		t.Errorf("log = %q, want the command arguments left out", out)
	}
}

// TestDispatcher_CanceledContextStaysBelowError pins the shutdown behavior: a
// canceled context is the bot stopping, not a service failure, and an ERROR
// record would be mirrored into a service chat that is closing right then.
//
// The error is produced by the real telegram provider instead of being built
// by hand: the guard only holds while the provider keeps the cancellation in
// its error chain, and a hand-built error would satisfy the assertion no
// matter what the provider actually returns.
func TestDispatcher_CanceledContextStaysBelowError(t *testing.T) {
	logOut := captureDefaultLogger(t)

	chat := &fakeChatController{err: buildCanceledReplyError(t)}
	d := NewDispatcher(chat, &fakeAdminController{}, testConfig())

	d.handleVoice(context.Background(), nil, voiceUpdate(testAllowedID, 1))

	out := logOut.String()
	if strings.Contains(out, "level=ERROR") {
		t.Errorf("log = %q, want the canceled context below ERROR", out)
	}

	if !strings.Contains(out, "level=WARN") || !strings.Contains(out, "shutting down") {
		t.Errorf("log = %q, want a WARN record marked as a shutdown", out)
	}

	// The same record carries the provider error text, which quotes the
	// request URL: the token must not ride along into the log.
	if strings.Contains(out, testBotToken) {
		t.Errorf("log = %q, want the bot token redacted", out)
	}
}

// buildCanceledReplyError reproduces the error a controller returns when the
// shutdown cancels a reply in flight: the real telegram provider called with
// an already canceled context, wrapped in the controller sentinel the way
// chat-controller wraps it.
func buildCanceledReplyError(t *testing.T) error {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider := telegramProvider.NewProvider(canceledBotAPI{}, providerConfig{}, unusedDoer{})

	err := provider.SendReply(ctx, testAllowedID, 1, "transcript")
	if err == nil {
		t.Fatal("SendReply() with a canceled context returned no error")
	}

	return fmt.Errorf("%w: %w", controllers.ErrSendReply, err)
}

// canceledBotAPI is a hand-written fake of the bot API the telegram provider
// consumes: a call fails the way net/http fails a request whose context is
// already done — a *url.Error carrying the request URL, token included, and
// wrapping the context error.
type canceledBotAPI struct{}

func (canceledBotAPI) GetFile(ctx context.Context, _ *bot.GetFileParams) (*tgmodels.File, error) {
	return nil, &url.Error{Op: http.MethodPost, URL: testAPIURL + "/getFile", Err: ctx.Err()}
}

func (canceledBotAPI) SendMessage(ctx context.Context, _ *bot.SendMessageParams) (*tgmodels.Message, error) {
	return nil, &url.Error{Op: http.MethodPost, URL: testAPIURL + "/sendMessage", Err: ctx.Err()}
}

// providerConfig is a hand-written fake of the config the telegram provider
// consumes.
type providerConfig struct{}

func (providerConfig) GetTelegramToken() string { return testBotToken }

func (providerConfig) GetTelegramAPITimeout() time.Duration { return testProviderTimeout }

func (providerConfig) GetTelegramDownloadTimeout() time.Duration { return testProviderTimeout }

// unusedDoer stands for the HTTP client the provider only uses to download
// files: sending a reply never reaches it.
type unusedDoer struct{}

func (unusedDoer) Do(*http.Request) (*http.Response, error) {
	return nil, errors.New("unexpected file download")
}
