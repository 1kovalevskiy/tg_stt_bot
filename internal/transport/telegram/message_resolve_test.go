package telegram

import (
	"context"
	"testing"

	tgmodels "github.com/go-telegram/bot/models"
)

func TestDispatcher_ForeignChatIsIgnored(t *testing.T) {
	t.Parallel()

	d, chat, admin := newTestDispatcher()

	voice := voiceUpdate(testForeignID, 1)
	if d.matchVoice(voice) {
		t.Error("matchVoice() = true for a foreign chat, want false")
	}

	d.handleVoice(context.Background(), nil, voice)

	note := videoNoteUpdate(testForeignID, 2)
	if d.matchVideoNote(note) {
		t.Error("matchVideoNote() = true for a foreign chat, want false")
	}

	d.handleVideoNote(context.Background(), nil, note)

	if len(chat.voice) != 0 || len(chat.videoNote) != 0 {
		t.Errorf("chat controller called with %+v / %+v, want no calls", chat.voice, chat.videoNote)
	}

	if len(admin.calls) != 0 {
		t.Errorf("admin controller called %+v, want no calls", admin.calls)
	}
}

func TestDispatcher_NonMessageUpdatesAreIgnored(t *testing.T) {
	t.Parallel()

	d, chat, admin := newTestDispatcher()

	updates := []*tgmodels.Update{
		nil,
		{},
		{Message: &tgmodels.Message{ID: 1, Chat: tgmodels.Chat{ID: testAllowedID}, Text: "просто текст"}},
	}

	for _, update := range updates {
		if d.matchVoice(update) || d.matchVideoNote(update) || d.matchAdminCommand(update) {
			t.Errorf("update %+v matched a handler, want no match", update)
		}

		d.handleVoice(context.Background(), nil, update)
		d.handleVideoNote(context.Background(), nil, update)
		d.handleAdminCommand(context.Background(), nil, update)
	}

	if len(chat.voice) != 0 || len(chat.videoNote) != 0 || len(admin.calls) != 0 {
		t.Errorf("controllers called for non-audio updates: %+v / %+v / %+v",
			chat.voice, chat.videoNote, admin.calls)
	}
}

// TestResolveAdminMessage_OnlyThePrivateAdminChat pins the second resolver: the
// admin matcher reads the admin id through it and nowhere else.
func TestResolveAdminMessage_OnlyThePrivateAdminChat(t *testing.T) {
	t.Parallel()

	d, _, _ := newTestDispatcher()

	tests := []struct {
		name    string
		update  *tgmodels.Update
		wantNil bool
	}{
		{name: "admin chat", update: textUpdate(testAdminID, "/status"), wantNil: false},
		{name: "allowed group chat", update: textUpdate(testAllowedID, "/status"), wantNil: true},
		{name: "foreign chat", update: textUpdate(testForeignID, "/status"), wantNil: true},
		{name: "nil update", update: nil, wantNil: true},
		{name: "update without a message", update: &tgmodels.Update{}, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := d.resolveAdminMessage(tt.update); (got == nil) != tt.wantNil {
				t.Errorf("resolveAdminMessage() = %+v, want nil: %v", got, tt.wantNil)
			}
		})
	}
}
