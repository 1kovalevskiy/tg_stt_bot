package chatController

import (
	"context"
	"io"
	"strings"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

// The fakes and fixtures below are shared by every test of this package; the
// tests themselves live next to the file they cover.

const (
	testChatID    int64 = -100123
	testMessageID       = 777
	testFileID          = "file-1"
	testFileSize  int64 = 1024
	// testTranscript is what the fake STT service recognizes on the happy path.
	testTranscript = "привет мир"
)

// sentReply records a single SendReply call.
type sentReply struct {
	chatID    int64
	messageID int
	text      string
}

// trackingBody is a ReadCloser that records whether Close was called and can
// fail on Close.
type trackingBody struct {
	io.Reader
	closed   bool
	closeErr error
}

func (b *trackingBody) Close() error {
	b.closed = true

	return b.closeErr
}

// fakeTelegram is a hand-written fake of the telegramProvider interface.
type fakeTelegram struct {
	body        *trackingBody
	gotFileIDs  []string
	downloadErr error

	replies []sentReply
	// replyCtxErrs is the state of the context each reply was sent with.
	replyCtxErrs []error
	replyErr     error
	// replyErrFrom is how many replies succeed before replyErr starts firing.
	replyErrFrom int
}

func newFakeTelegram(content string) *fakeTelegram {
	return &fakeTelegram{body: &trackingBody{Reader: strings.NewReader(content)}}
}

func (f *fakeTelegram) DownloadFile(_ context.Context, fileID string) (io.ReadCloser, error) {
	f.gotFileIDs = append(f.gotFileIDs, fileID)
	if f.downloadErr != nil {
		return nil, f.downloadErr
	}

	return f.body, nil
}

func (f *fakeTelegram) SendReply(ctx context.Context, chatID int64, replyToMessageID int, text string) error {
	f.replies = append(f.replies, sentReply{chatID: chatID, messageID: replyToMessageID, text: text})
	f.replyCtxErrs = append(f.replyCtxErrs, ctx.Err())

	if f.replyErr != nil && len(f.replies) > f.replyErrFrom {
		return f.replyErr
	}

	return nil
}

// fakeSTT is a hand-written fake of the sttProvider interface.
type fakeSTT struct {
	calls        int
	gotFilenames []string
	gotAudio     string

	text string
	err  error
}

func (f *fakeSTT) TranscribeAudio(_ context.Context, audio io.Reader, filename string) (string, error) {
	f.calls++
	f.gotFilenames = append(f.gotFilenames, filename)

	content, err := io.ReadAll(audio)
	if err != nil {
		return "", err
	}

	f.gotAudio = string(content)

	if f.err != nil {
		return "", f.err
	}

	return f.text, nil
}

func testAudio() models.IncomingAudio {
	return models.IncomingAudio{
		ChatID:    testChatID,
		MessageID: testMessageID,
		FileID:    testFileID,
		FileSize:  testFileSize,
	}
}
