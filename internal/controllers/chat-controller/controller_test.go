package chatController

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/controllers"
	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
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
	return models.IncomingAudio{ChatID: -100123, MessageID: 777, FileID: "file-1", FileSize: 1024}
}

func TestHandleVoice_Success(t *testing.T) {
	telegram := newFakeTelegram("audio-bytes")
	stt := &fakeSTT{text: "привет мир"}
	controller := NewController(telegram, stt)

	if err := controller.HandleVoice(context.Background(), testAudio()); err != nil {
		t.Fatalf("HandleVoice() unexpected error: %v", err)
	}

	if len(telegram.gotFileIDs) != 1 || telegram.gotFileIDs[0] != "file-1" {
		t.Errorf("DownloadFile called with %v, want [file-1]", telegram.gotFileIDs)
	}

	if stt.gotAudio != "audio-bytes" {
		t.Errorf("STT got audio %q, want %q", stt.gotAudio, "audio-bytes")
	}

	if len(stt.gotFilenames) != 1 || stt.gotFilenames[0] != models.VoiceFilename {
		t.Errorf("STT got filenames %v, want [%s]", stt.gotFilenames, models.VoiceFilename)
	}

	if len(telegram.replies) != 1 {
		t.Fatalf("SendReply called %d times, want 1", len(telegram.replies))
	}

	want := sentReply{chatID: -100123, messageID: 777, text: "привет мир"}
	if telegram.replies[0] != want {
		t.Errorf("reply = %+v, want %+v", telegram.replies[0], want)
	}

	if !telegram.body.closed {
		t.Error("downloaded file was not closed")
	}
}

func TestHandleVideoNote_Success(t *testing.T) {
	telegram := newFakeTelegram("video-bytes")
	stt := &fakeSTT{text: "текст из кружочка"}
	controller := NewController(telegram, stt)

	if err := controller.HandleVideoNote(context.Background(), testAudio()); err != nil {
		t.Fatalf("HandleVideoNote() unexpected error: %v", err)
	}

	if len(stt.gotFilenames) != 1 || stt.gotFilenames[0] != models.VideoNoteFilename {
		t.Errorf("STT got filenames %v, want [%s]", stt.gotFilenames, models.VideoNoteFilename)
	}

	if len(telegram.replies) != 1 || telegram.replies[0].text != "текст из кружочка" {
		t.Errorf("replies = %+v, want single reply with the transcript", telegram.replies)
	}

	if !telegram.body.closed {
		t.Error("downloaded file was not closed")
	}
}

func TestHandleVoice_LongTextSplitIntoSeveralMessages(t *testing.T) {
	longText := strings.TrimSpace(strings.Repeat("слово ", 1000)) // 5999 UTF-16 units
	telegram := newFakeTelegram("audio-bytes")
	stt := &fakeSTT{text: longText}
	controller := NewController(telegram, stt)

	if err := controller.HandleVoice(context.Background(), testAudio()); err != nil {
		t.Fatalf("HandleVoice() unexpected error: %v", err)
	}

	if len(telegram.replies) < 2 {
		t.Fatalf("SendReply called %d times, want at least 2 for a long transcript", len(telegram.replies))
	}

	var joined strings.Builder

	for i, reply := range telegram.replies {
		if reply.chatID != -100123 || reply.messageID != 777 {
			t.Errorf("reply %d = %+v, want chat -100123 and reply to 777", i, reply)
		}

		if i > 0 {
			joined.WriteString(" ")
		}

		joined.WriteString(reply.text)
	}

	if joined.String() != longText {
		t.Error("concatenated chunks do not reconstruct the transcript")
	}
}

func TestHandleVoice_EmptyTranscription(t *testing.T) {
	telegram := newFakeTelegram("audio-bytes")
	stt := &fakeSTT{text: "  \n\t "}
	controller := NewController(telegram, stt)

	if err := controller.HandleVoice(context.Background(), testAudio()); err != nil {
		t.Fatalf("HandleVoice() unexpected error: %v", err)
	}

	if len(telegram.replies) != 1 || telegram.replies[0].text != models.MsgNoSpeech {
		t.Errorf("replies = %+v, want single %q reply", telegram.replies, models.MsgNoSpeech)
	}

	if !telegram.body.closed {
		t.Error("downloaded file was not closed")
	}
}

func TestHandleVoice_FileTooLarge(t *testing.T) {
	telegram := newFakeTelegram("audio-bytes")
	stt := &fakeSTT{text: "unused"}
	controller := NewController(telegram, stt)

	audio := testAudio()
	audio.FileSize = models.TelegramMaxFileSize + 1

	if err := controller.HandleVoice(context.Background(), audio); err != nil {
		t.Fatalf("HandleVoice() unexpected error: %v", err)
	}

	if len(telegram.gotFileIDs) != 0 {
		t.Errorf("DownloadFile called %v, want no download for an oversized file", telegram.gotFileIDs)
	}

	if stt.calls != 0 {
		t.Errorf("TranscribeAudio called %d times, want 0 for an oversized file", stt.calls)
	}

	if len(telegram.replies) != 1 || telegram.replies[0].text != models.MsgFileTooLarge {
		t.Errorf("replies = %+v, want single %q reply", telegram.replies, models.MsgFileTooLarge)
	}
}

func TestHandleVoice_DownloadError(t *testing.T) {
	downloadErr := errors.New("connection refused")
	telegram := newFakeTelegram("audio-bytes")
	telegram.downloadErr = downloadErr
	stt := &fakeSTT{text: "unused"}
	controller := NewController(telegram, stt)

	err := controller.HandleVoice(context.Background(), testAudio())
	if !errors.Is(err, controllers.ErrDownloadAudio) {
		t.Fatalf("HandleVoice() error = %v, want errors.Is controllers.ErrDownloadAudio", err)
	}

	if !errors.Is(err, downloadErr) {
		t.Errorf("HandleVoice() error = %v, want it to wrap the provider error", err)
	}

	if stt.calls != 0 {
		t.Errorf("TranscribeAudio called %d times, want 0 after a download failure", stt.calls)
	}

	if len(telegram.replies) != 1 || telegram.replies[0].text != models.MsgTranscribeFailed {
		t.Errorf("replies = %+v, want single %q reply", telegram.replies, models.MsgTranscribeFailed)
	}
}

func TestHandleVoice_TranscribeError(t *testing.T) {
	sttErr := errors.New("stt returned unexpected status")
	telegram := newFakeTelegram("audio-bytes")
	stt := &fakeSTT{err: sttErr}
	controller := NewController(telegram, stt)

	err := controller.HandleVoice(context.Background(), testAudio())
	if !errors.Is(err, controllers.ErrTranscribeAudio) {
		t.Fatalf("HandleVoice() error = %v, want errors.Is controllers.ErrTranscribeAudio", err)
	}

	if !errors.Is(err, sttErr) {
		t.Errorf("HandleVoice() error = %v, want it to wrap the provider error", err)
	}

	if len(telegram.replies) != 1 || telegram.replies[0].text != models.MsgTranscribeFailed {
		t.Errorf("replies = %+v, want single %q reply", telegram.replies, models.MsgTranscribeFailed)
	}

	if !telegram.body.closed {
		t.Error("downloaded file was not closed after an STT failure")
	}
}

func TestHandleVoice_ReplyError(t *testing.T) {
	replyErr := errors.New("chat not found")
	telegram := newFakeTelegram("audio-bytes")
	telegram.replyErr = replyErr
	stt := &fakeSTT{text: "привет мир"}
	controller := NewController(telegram, stt)

	err := controller.HandleVoice(context.Background(), testAudio())
	if !errors.Is(err, controllers.ErrSendReply) {
		t.Fatalf("HandleVoice() error = %v, want errors.Is controllers.ErrSendReply", err)
	}

	if !errors.Is(err, replyErr) {
		t.Errorf("HandleVoice() error = %v, want it to wrap the provider error", err)
	}
}

func TestHandleVoice_TranscribeAndReplyErrorsAreJoined(t *testing.T) {
	sttErr := errors.New("stt down")
	replyErr := errors.New("chat not found")
	telegram := newFakeTelegram("audio-bytes")
	telegram.replyErr = replyErr
	stt := &fakeSTT{err: sttErr}
	controller := NewController(telegram, stt)

	err := controller.HandleVoice(context.Background(), testAudio())
	if !errors.Is(err, controllers.ErrTranscribeAudio) {
		t.Errorf("HandleVoice() error = %v, want errors.Is controllers.ErrTranscribeAudio", err)
	}

	if !errors.Is(err, controllers.ErrSendReply) {
		t.Errorf("HandleVoice() error = %v, want errors.Is controllers.ErrSendReply", err)
	}
}

func TestHandleVoice_FileSizeBoundary(t *testing.T) {
	tests := []struct {
		name          string
		fileSize      int64
		wantTranscibe bool
		wantReply     string
	}{
		{
			name: "exactly at the limit is still served",
			// getFile does serve a file of exactly this size.
			fileSize:      models.TelegramMaxFileSize,
			wantTranscibe: true,
			wantReply:     "привет мир",
		},
		{
			// file_size is an optional Bot API field: a missing one must not
			// be read as "empty file" and must not block the transcription.
			name:          "unknown size is transcribed",
			fileSize:      0,
			wantTranscibe: true,
			wantReply:     "привет мир",
		},
		{
			name:          "one byte over the limit is rejected",
			fileSize:      models.TelegramMaxFileSize + 1,
			wantTranscibe: false,
			wantReply:     models.MsgFileTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telegram := newFakeTelegram("audio-bytes")
			stt := &fakeSTT{text: "привет мир"}
			controller := NewController(telegram, stt)

			audio := testAudio()
			audio.FileSize = tt.fileSize

			if err := controller.HandleVoice(context.Background(), audio); err != nil {
				t.Fatalf("HandleVoice() unexpected error: %v", err)
			}

			if got := stt.calls > 0; got != tt.wantTranscibe {
				t.Errorf("TranscribeAudio called = %v, want %v for size %d", got, tt.wantTranscibe, tt.fileSize)
			}

			if len(telegram.replies) != 1 || telegram.replies[0].text != tt.wantReply {
				t.Errorf("replies = %+v, want single %q reply", telegram.replies, tt.wantReply)
			}
		})
	}
}

func TestHandleVoice_ChunkFailureNotifiesTheUser(t *testing.T) {
	replyErr := errors.New("too many requests")
	longText := strings.TrimSpace(strings.Repeat("слово ", 1000))

	telegram := newFakeTelegram("audio-bytes")
	telegram.replyErr = replyErr
	// The first chunk goes through, the rest fail: Telegram rate-limits rapid
	// multi-chunk sends.
	telegram.replyErrFrom = 1
	stt := &fakeSTT{text: longText}
	controller := NewController(telegram, stt)

	err := controller.HandleVoice(context.Background(), testAudio())
	if !errors.Is(err, controllers.ErrSendReply) {
		t.Fatalf("HandleVoice() error = %v, want errors.Is controllers.ErrSendReply", err)
	}

	if len(telegram.replies) < 3 {
		t.Fatalf("SendReply called %d times, want the failed chunk and a failure notice", len(telegram.replies))
	}

	// A truncated transcript with no notice would look like the whole answer.
	last := telegram.replies[len(telegram.replies)-1]
	if last.text != models.MsgTranscribeFailed {
		t.Errorf("last reply = %q, want %q after a chunk failure", last.text, models.MsgTranscribeFailed)
	}
}

func TestHandleVoice_FailureNoticeSurvivesCanceledContext(t *testing.T) {
	telegram := newFakeTelegram("audio-bytes")
	stt := &fakeSTT{err: context.Canceled}
	controller := NewController(telegram, stt)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := controller.HandleVoice(ctx, testAudio())
	if !errors.Is(err, controllers.ErrTranscribeAudio) {
		t.Fatalf("HandleVoice() error = %v, want errors.Is controllers.ErrTranscribeAudio", err)
	}

	if len(telegram.replies) != 1 || telegram.replies[0].text != models.MsgTranscribeFailed {
		t.Fatalf("replies = %+v, want single %q reply", telegram.replies, models.MsgTranscribeFailed)
	}

	// On shutdown the incoming context is already canceled: reusing it would
	// leave the user without any answer at all.
	if telegram.replyCtxErrs[0] != nil {
		t.Errorf("failure notice sent with context error %v, want a live context", telegram.replyCtxErrs[0])
	}
}

func TestHandleVoice_CloseErrorDoesNotFailTheScenario(t *testing.T) {
	telegram := newFakeTelegram("audio-bytes")
	telegram.body.closeErr = errors.New("connection reset")
	stt := &fakeSTT{text: "привет мир"}
	controller := NewController(telegram, stt)

	// The transcript is already sent when the body is closed: a close failure
	// is logged, not turned into a user-visible error.
	if err := controller.HandleVoice(context.Background(), testAudio()); err != nil {
		t.Fatalf("HandleVoice() unexpected error: %v", err)
	}

	if !telegram.body.closed {
		t.Error("downloaded file was not closed")
	}

	if len(telegram.replies) != 1 || telegram.replies[0].text != "привет мир" {
		t.Errorf("replies = %+v, want the transcript delivered", telegram.replies)
	}
}

func TestHandleVoice_ReplyErrorOnExpectedOutcomes(t *testing.T) {
	replyErr := errors.New("chat not found")

	tests := []struct {
		name     string
		fileSize int64
		text     string
	}{
		{name: "file too large", fileSize: models.TelegramMaxFileSize + 1, text: "unused"},
		{name: "no speech", fileSize: 1024, text: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telegram := newFakeTelegram("audio-bytes")
			telegram.replyErr = replyErr
			stt := &fakeSTT{text: tt.text}
			controller := NewController(telegram, stt)

			audio := testAudio()
			audio.FileSize = tt.fileSize

			err := controller.HandleVoice(context.Background(), audio)
			if !errors.Is(err, controllers.ErrSendReply) {
				t.Fatalf("HandleVoice() error = %v, want errors.Is controllers.ErrSendReply", err)
			}

			if !errors.Is(err, replyErr) {
				t.Errorf("HandleVoice() error = %v, want it to wrap the provider error", err)
			}
		})
	}
}
