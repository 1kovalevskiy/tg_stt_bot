package telegram

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// The fakes and helpers below are shared by every test of this package; the
// tests themselves live next to the file they cover.

const (
	testToken = "1234567890:AAF-secret-token-value"
	// testAPIURL is the Bot API endpoint prefix: it carries the token, which
	// is exactly why the errors quoting it have to be redacted.
	testAPIURL = "https://api.telegram.org/bot" + testToken
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

// fakeBotAPI is a hand-written fake of the botAPI interface. A call made with
// an already canceled or expired context fails the way the library fails it,
// so a test cannot assert a chain the real call can never produce.
type fakeBotAPI struct {
	getFileCtx    context.Context
	getFileParams *bot.GetFileParams
	getFileResult *tgmodels.File
	getFileErr    error
	// afterGetFile runs once GetFile has answered: it lets a test cancel the
	// caller's context between the metadata call and the download.
	afterGetFile func()

	sendCtx    []context.Context
	sendParams []*bot.SendMessageParams
	sendErr    error
}

func (f *fakeBotAPI) GetFile(ctx context.Context, params *bot.GetFileParams) (*tgmodels.File, error) {
	f.getFileCtx = ctx
	f.getFileParams = params

	if err := ctx.Err(); err != nil {
		return nil, newRequestError(testAPIURL+"/getFile", err)
	}

	if f.afterGetFile != nil {
		f.afterGetFile()
	}

	if f.getFileErr != nil {
		return nil, f.getFileErr
	}

	return f.getFileResult, nil
}

func (f *fakeBotAPI) SendMessage(ctx context.Context, params *bot.SendMessageParams) (*tgmodels.Message, error) {
	f.sendCtx = append(f.sendCtx, ctx)
	f.sendParams = append(f.sendParams, params)

	if err := ctx.Err(); err != nil {
		return nil, newRequestError(testAPIURL+"/sendMessage", err)
	}

	if f.sendErr != nil {
		return nil, f.sendErr
	}

	return &tgmodels.Message{}, nil
}

// fakeDoer is a hand-written fake of the httpDoer interface. Like the bot API
// fake, it fails a request whose context is already done.
type fakeDoer struct {
	gotURL string
	gotCtx context.Context
	resp   *http.Response
	err    error
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.gotURL = req.URL.String()
	f.gotCtx = req.Context()

	if err := req.Context().Err(); err != nil {
		return nil, newRequestError(req.URL.String(), err)
	}

	if f.err != nil {
		return nil, f.err
	}

	return f.resp, nil
}

// newRequestError builds the error net/http returns for a failed request: a
// *url.Error carrying the request URL — the bot token included — and wrapping
// the cause, which is how a canceled context reaches the provider.
func newRequestError(rawURL string, cause error) error {
	return &url.Error{Op: http.MethodPost, URL: rawURL, Err: cause}
}

// assertCanceledError checks both properties a canceled provider call has to
// hold at once: the shutdown stays matchable with errors.Is, so the transport
// keeps it below ERROR, and the token stays out of the message.
func assertCanceledError(t *testing.T, err error, sentinel error) {
	t.Helper()

	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want errors.Is %v", err, sentinel)
	}

	if !errors.Is(err, providers.ErrTelegramRequestCanceled) {
		t.Errorf("error = %v, want errors.Is providers.ErrTelegramRequestCanceled", err)
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want it to wrap context.Canceled", err)
	}

	if strings.Contains(err.Error(), testToken) {
		t.Errorf("error %q contains the bot token", err.Error())
	}

	if !strings.Contains(err.Error(), models.RedactedToken) {
		t.Errorf("error %q does not contain the redaction marker", err.Error())
	}
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
