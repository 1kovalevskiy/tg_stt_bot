package chatController

import (
	"context"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

func TestHandleVideoNote_Success(t *testing.T) {
	t.Parallel()

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
