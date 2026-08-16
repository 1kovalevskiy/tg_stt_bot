package telegram

import (
	"context"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

func TestDispatcher_VideoNoteFromAllowedChat(t *testing.T) {
	t.Parallel()

	d, chat, _ := newTestDispatcher()
	update := videoNoteUpdate(testAllowedID, 7)

	if !d.matchVideoNote(update) {
		t.Fatal("matchVideoNote() = false, want true for a video note from an allowed chat")
	}

	if d.matchVoice(update) {
		t.Error("matchVoice() = true for a video note, want false")
	}

	d.handleVideoNote(context.Background(), nil, update)

	want := models.IncomingAudio{
		ChatID:    testAllowedID,
		MessageID: 7,
		FileID:    "video-note-file-id",
		FileSize:  4096,
	}

	if len(chat.videoNote) != 1 || chat.videoNote[0] != want {
		t.Errorf("HandleVideoNote calls = %+v, want [%+v]", chat.videoNote, want)
	}
}

// TestDispatcher_VideoNoteMatcherIgnoresOtherAudio is the mirror of the voice
// case: the two kinds share one helper and must not collapse into one.
func TestDispatcher_VideoNoteMatcherIgnoresOtherAudio(t *testing.T) {
	t.Parallel()

	d, chat, _ := newTestDispatcher()
	update := voiceUpdate(testAllowedID, 4)

	if d.matchVideoNote(update) {
		t.Error("matchVideoNote() = true for a voice message, want false")
	}

	d.handleVideoNote(context.Background(), nil, update)

	if len(chat.videoNote) != 0 {
		t.Errorf("HandleVideoNote calls = %+v, want no calls for a voice message", chat.videoNote)
	}
}
