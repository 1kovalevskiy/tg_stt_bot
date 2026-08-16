package telegram

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	tgmodels "github.com/go-telegram/bot/models"
)

// The fakes, fixtures and helpers below are shared by every test of this
// package; the tests themselves live next to the file they cover.

const (
	testAdminID   int64 = 555
	testAllowedID int64 = -100123
	testForeignID int64 = -100999
	testBotToken        = "123456:AAHtest-token"
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

// fakeConfig is a hand-written fake of the config interfaces this package
// consumes: the chat access rules and the bot token.
type fakeConfig struct {
	adminID int64
	allowed []int64
	token   string
}

func (f fakeConfig) GetTelegramAdminID() int64 { return f.adminID }

func (f fakeConfig) GetTelegramAllowedChats() []int64 { return f.allowed }

func (f fakeConfig) GetTelegramToken() string { return f.token }

func testConfig() fakeConfig {
	return fakeConfig{adminID: testAdminID, allowed: []int64{testAllowedID}, token: testBotToken}
}

func newTestDispatcher() (*Dispatcher, *fakeChatController, *fakeAdminController) {
	chat := &fakeChatController{}
	admin := &fakeAdminController{}

	return NewDispatcher(chat, admin, testConfig()), chat, admin
}

// captureDefaultLogger redirects the global logger into a buffer for the
// duration of the test: controller failures are reported through it, and a
// canceled context has to stay below ERROR so it never reaches the service chat.
// A test using it touches global state and therefore cannot run in parallel.
func captureDefaultLogger(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	previous := slog.Default()

	t.Cleanup(func() { slog.SetDefault(previous) })
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	return buf
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
