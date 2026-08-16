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

// trackingBody is a ReadCloser that records whether Close was called.
type trackingBody struct {
	io.Reader
	closed bool
}

func (b *trackingBody) Close() error {
	b.closed = true

	return nil
}

// fakeTelegram is a hand-written fake of the telegramProvider interface.
type fakeTelegram struct {
	body        *trackingBody
	gotFileIDs  []string
	downloadErr error

	replies  []sentReply
	replyErr error
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

func (f *fakeTelegram) SendReply(_ context.Context, chatID int64, replyToMessageID int, text string) error {
	f.replies = append(f.replies, sentReply{chatID: chatID, messageID: replyToMessageID, text: text})
	if f.replyErr != nil {
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

func (f *fakeSTT) Transcribe(_ context.Context, audio io.Reader, filename string) (string, error) {
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

	if len(stt.gotFilenames) != 1 || stt.gotFilenames[0] != voiceFilename {
		t.Errorf("STT got filenames %v, want [%s]", stt.gotFilenames, voiceFilename)
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

	if len(stt.gotFilenames) != 1 || stt.gotFilenames[0] != videoNoteFilename {
		t.Errorf("STT got filenames %v, want [%s]", stt.gotFilenames, videoNoteFilename)
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

	if len(telegram.replies) != 1 || telegram.replies[0].text != msgNoSpeech {
		t.Errorf("replies = %+v, want single %q reply", telegram.replies, msgNoSpeech)
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
	audio.FileSize = maxFileSize + 1

	if err := controller.HandleVoice(context.Background(), audio); err != nil {
		t.Fatalf("HandleVoice() unexpected error: %v", err)
	}

	if len(telegram.gotFileIDs) != 0 {
		t.Errorf("DownloadFile called %v, want no download for an oversized file", telegram.gotFileIDs)
	}

	if stt.calls != 0 {
		t.Errorf("Transcribe called %d times, want 0 for an oversized file", stt.calls)
	}

	if len(telegram.replies) != 1 || telegram.replies[0].text != msgFileTooLarge {
		t.Errorf("replies = %+v, want single %q reply", telegram.replies, msgFileTooLarge)
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
		t.Errorf("Transcribe called %d times, want 0 after a download failure", stt.calls)
	}

	if len(telegram.replies) != 1 || telegram.replies[0].text != msgTranscribeFailed {
		t.Errorf("replies = %+v, want single %q reply", telegram.replies, msgTranscribeFailed)
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

	if len(telegram.replies) != 1 || telegram.replies[0].text != msgTranscribeFailed {
		t.Errorf("replies = %+v, want single %q reply", telegram.replies, msgTranscribeFailed)
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
