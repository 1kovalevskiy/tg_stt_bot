package chatController

import (
	"context"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

func TestHandleVoice_Success(t *testing.T) {
	t.Parallel()

	telegram := newFakeTelegram("audio-bytes")
	stt := &fakeSTT{text: testTranscript}
	controller := NewController(telegram, stt)

	if err := controller.HandleVoice(context.Background(), testAudio()); err != nil {
		t.Fatalf("HandleVoice() unexpected error: %v", err)
	}

	if len(telegram.gotFileIDs) != 1 || telegram.gotFileIDs[0] != testFileID {
		t.Errorf("DownloadFile called with %v, want [%s]", telegram.gotFileIDs, testFileID)
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

	want := sentReply{chatID: testChatID, messageID: testMessageID, text: testTranscript}
	if telegram.replies[0] != want {
		t.Errorf("reply = %+v, want %+v", telegram.replies[0], want)
	}

	if !telegram.body.closed {
		t.Error("downloaded file was not closed")
	}
}
