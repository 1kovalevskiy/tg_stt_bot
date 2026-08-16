package app

import (
	"context"
	"errors"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	tgmodels "github.com/go-telegram/bot/models"
)

// fakeChatController is a hand-written fake of the chatHandler interface.
type fakeChatController struct {
	voice     []models.IncomingAudio
	videoNote []models.IncomingAudio

	err error
}

func (f *fakeChatController) HandleVoice(_ context.Context, audio models.IncomingAudio) error {
	f.voice = append(f.voice, audio)

	return f.err
}

func (f *fakeChatController) HandleVideoNote(_ context.Context, audio models.IncomingAudio) error {
	f.videoNote = append(f.videoNote, audio)

	return f.err
}

// adminCall records a single HandleCommand call.
type adminCall struct {
	chatID  int64
	command string
}

// fakeAdminController is a hand-written fake of the adminHandler interface.
type fakeAdminController struct {
	calls []adminCall

	err error
}

func (f *fakeAdminController) HandleCommand(_ context.Context, chatID int64, command string) error {
	f.calls = append(f.calls, adminCall{chatID: chatID, command: command})

	return f.err
}

const (
	testAdminID   int64 = 555
	testAllowedID int64 = -100123
	testForeignID int64 = -100999
)

func newTestDispatcher() (*dispatcher, *fakeChatController, *fakeAdminController) {
	chat := &fakeChatController{}
	admin := &fakeAdminController{}

	return newDispatcher(chat, admin, testAdminID, []int64{testAllowedID}), chat, admin
}

func voiceUpdate(chatID int64, messageID int) *tgmodels.Update {
	return &tgmodels.Update{
		Message: &tgmodels.Message{
			ID:    messageID,
			Chat:  tgmodels.Chat{ID: chatID},
			Voice: &tgmodels.Voice{FileID: "voice-file-id", FileSize: 2048},
		},
	}
}

func videoNoteUpdate(chatID int64, messageID int) *tgmodels.Update {
	return &tgmodels.Update{
		Message: &tgmodels.Message{
			ID:        messageID,
			Chat:      tgmodels.Chat{ID: chatID},
			VideoNote: &tgmodels.VideoNote{FileID: "video-note-file-id", FileSize: 4096},
		},
	}
}

func textUpdate(chatID int64, text string) *tgmodels.Update {
	return &tgmodels.Update{
		Message: &tgmodels.Message{
			ID:   1,
			Chat: tgmodels.Chat{ID: chatID},
			Text: text,
		},
	}
}

func TestDispatcher_VoiceFromAllowedChat(t *testing.T) {
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

func TestDispatcher_VideoNoteFromAllowedChat(t *testing.T) {
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

func TestDispatcher_ForeignChatIsIgnored(t *testing.T) {
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

func TestDispatcher_AdminPrivateChatVoiceGoesToChatController(t *testing.T) {
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

func TestDispatcher_AdminCommand(t *testing.T) {
	d, chat, admin := newTestDispatcher()
	update := textUpdate(testAdminID, "/status")

	if !d.matchAdminCommand(update) {
		t.Fatal("matchAdminCommand() = false, want true for a command from the admin")
	}

	d.handleAdminCommand(context.Background(), nil, update)

	want := adminCall{chatID: testAdminID, command: "/status"}
	if len(admin.calls) != 1 || admin.calls[0] != want {
		t.Errorf("admin calls = %+v, want [%+v]", admin.calls, want)
	}

	if len(chat.voice) != 0 || len(chat.videoNote) != 0 {
		t.Errorf("chat controller called with %+v / %+v, want no calls", chat.voice, chat.videoNote)
	}
}

func TestDispatcher_CommandFromNonAdminIsIgnored(t *testing.T) {
	d, _, admin := newTestDispatcher()

	tests := []struct {
		name   string
		update *tgmodels.Update
	}{
		{name: "allowed group chat", update: textUpdate(testAllowedID, "/status")},
		{name: "foreign chat", update: textUpdate(testForeignID, "/status")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if d.matchAdminCommand(tt.update) {
				t.Error("matchAdminCommand() = true, want false outside the admin private chat")
			}

			d.handleAdminCommand(context.Background(), nil, tt.update)

			if len(admin.calls) != 0 {
				t.Errorf("admin calls = %+v, want no calls", admin.calls)
			}
		})
	}
}

func TestDispatcher_PlainTextFromAdminIsNotACommand(t *testing.T) {
	d, _, admin := newTestDispatcher()

	tests := []struct {
		name string
		text string
	}{
		{name: "plain text", text: "привет"},
		{name: "empty text", text: ""},
		{name: "slash inside text", text: "смотри /status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := textUpdate(testAdminID, tt.text)

			if d.matchAdminCommand(update) {
				t.Error("matchAdminCommand() = true, want false for a non-command message")
			}

			d.handleAdminCommand(context.Background(), nil, update)

			if len(admin.calls) != 0 {
				t.Errorf("admin calls = %+v, want no calls", admin.calls)
			}
		})
	}
}

func TestDispatcher_CommandWithLeadingSpaceIsAccepted(t *testing.T) {
	d, _, admin := newTestDispatcher()
	update := textUpdate(testAdminID, "  /chats  ")

	if !d.matchAdminCommand(update) {
		t.Fatal("matchAdminCommand() = false, want true for a padded command")
	}

	d.handleAdminCommand(context.Background(), nil, update)

	if len(admin.calls) != 1 || admin.calls[0].command != "  /chats  " {
		t.Errorf("admin calls = %+v, want the raw command text passed to the controller", admin.calls)
	}
}

func TestDispatcher_NonMessageUpdatesAreIgnored(t *testing.T) {
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

func TestDispatcher_ControllerErrorsAreSwallowed(t *testing.T) {
	chat := &fakeChatController{err: errors.New("stt is down")}
	admin := &fakeAdminController{err: errors.New("chat not found")}
	d := newDispatcher(chat, admin, testAdminID, []int64{testAllowedID})

	// A failing controller must not panic the worker: the error is logged
	// and the update is dropped.
	d.handleVoice(context.Background(), nil, voiceUpdate(testAllowedID, 1))
	d.handleVideoNote(context.Background(), nil, videoNoteUpdate(testAllowedID, 2))
	d.handleAdminCommand(context.Background(), nil, textUpdate(testAdminID, "/status"))

	if len(chat.voice) != 1 || len(chat.videoNote) != 1 || len(admin.calls) != 1 {
		t.Errorf("controllers called %+v / %+v / %+v, want one call each",
			chat.voice, chat.videoNote, admin.calls)
	}
}

func TestRedactToken(t *testing.T) {
	t.Parallel()

	const token = "123456:AAHfake-token"

	tests := []struct {
		name    string
		message string
		token   string
		want    string
	}{
		{
			name:    "token in url is redacted",
			message: `Get "https://api.telegram.org/bot123456:AAHfake-token/getMe": timeout`,
			token:   token,
			want:    `Get "https://api.telegram.org/bot[REDACTED]/getMe": timeout`,
		},
		{
			name:    "several occurrences",
			message: token + " and " + token,
			token:   token,
			want:    "[REDACTED] and [REDACTED]",
		},
		{
			name:    "no token in message",
			message: "error call getMe, unauthorized",
			token:   token,
			want:    "error call getMe, unauthorized",
		},
		{
			name:    "empty token leaves message intact",
			message: "some error",
			token:   "",
			want:    "some error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := redactToken(tt.message, tt.token)
			if got != tt.want {
				t.Errorf("redactToken() = %q, want %q", got, tt.want)
			}

			if tt.token != "" && got != tt.want {
				t.Errorf("redactToken() leaked the token: %q", got)
			}
		})
	}
}
