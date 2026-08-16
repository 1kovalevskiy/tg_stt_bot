package telegram

import (
	"context"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

func TestDispatcher_VoiceFromAllowedChat(t *testing.T) {
	t.Parallel()

	d, chat, admin := newTestDispatcher()
	update := voiceUpdate(testAllowedID, 42)

	if !d.matchVoice(update) {
		t.Fatal("matchVoice() = false, want true for a voice from an allowed chat")
	}

	d.handleVoice(context.Background(), nil, update)

	want := models.IncomingAudio{
		ChatID:    testAllowedID,
		MessageID: 42,
		FileID:    "voice-file-id",
		FileSize:  2048,
	}

	if len(chat.voice) != 1 || chat.voice[0] != want {
		t.Errorf("HandleVoice calls = %+v, want [%+v]", chat.voice, want)
	}

	if len(admin.calls) != 0 {
		t.Errorf("admin controller called %+v, want no calls", admin.calls)
	}
}

func TestDispatcher_AdminPrivateChatVoiceGoesToChatController(t *testing.T) {
	t.Parallel()

	d, chat, admin := newTestDispatcher()
	update := voiceUpdate(testAdminID, 9)

	if !d.matchVoice(update) {
		t.Fatal("matchVoice() = false, want true for a voice in the admin private chat")
	}

	d.handleVoice(context.Background(), nil, update)

	if len(chat.voice) != 1 || chat.voice[0].ChatID != testAdminID {
		t.Errorf("HandleVoice calls = %+v, want one call for the admin chat", chat.voice)
	}

	if len(admin.calls) != 0 {
		t.Errorf("admin controller called %+v, want no calls for audio", admin.calls)
	}
}

// TestDispatcher_VoiceMatcherIgnoresOtherAudio pins that the shared audio
// helper keeps the two kinds apart: a matcher accepting both would send video
// notes to HandleVoice.
func TestDispatcher_VoiceMatcherIgnoresOtherAudio(t *testing.T) {
	t.Parallel()

	d, chat, _ := newTestDispatcher()
	update := videoNoteUpdate(testAllowedID, 3)

	if d.matchVoice(update) {
		t.Error("matchVoice() = true for a video note, want false")
	}

	d.handleVoice(context.Background(), nil, update)

	if len(chat.voice) != 0 {
		t.Errorf("HandleVoice calls = %+v, want no calls for a video note", chat.voice)
	}
}
