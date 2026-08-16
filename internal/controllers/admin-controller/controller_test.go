package adminController

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// sentMessage records a single SendMessage call.
type sentMessage struct {
	chatID int64
	text   string
}

// fakeTelegram is a hand-written fake of the telegramProvider interface.
type fakeTelegram struct {
	messages []sentMessage
	sendErr  error
}

func (f *fakeTelegram) SendMessage(_ context.Context, chatID int64, text string) error {
	f.messages = append(f.messages, sentMessage{chatID: chatID, text: text})
	if f.sendErr != nil {
		return f.sendErr
	}

	return nil
}

// fakeSTT is a hand-written fake of the sttProvider interface.
type fakeSTT struct {
	calls  int
	health string
	err    error
}

func (f *fakeSTT) Health(_ context.Context) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}

	return f.health, nil
}

// fakeConfig is a hand-written fake of the configProvider interface.
type fakeConfig struct {
	allowed []int64
}

func (f *fakeConfig) GetTelegramAllowedChats() []int64 {
	return f.allowed
}

const adminChatID int64 = 555

func TestHandleCommand_StatusHealthySTT(t *testing.T) {
	telegram := &fakeTelegram{}
	stt := &fakeSTT{health: `{"status":"ok"}`}
	controller := NewController(telegram, stt, &fakeConfig{})

	if err := controller.HandleCommand(context.Background(), adminChatID, "/status"); err != nil {
		t.Fatalf("HandleCommand() unexpected error: %v", err)
	}

	if stt.calls != 1 {
		t.Errorf("Health called %d times, want 1", stt.calls)
	}

	want := sentMessage{chatID: adminChatID, text: statusPrefix + `{"status":"ok"}`}
	if len(telegram.messages) != 1 || telegram.messages[0] != want {
		t.Errorf("messages = %+v, want [%+v]", telegram.messages, want)
	}
}

func TestHandleCommand_StatusDeadSTT(t *testing.T) {
	healthErr := errors.New("connection refused")
	telegram := &fakeTelegram{}
	stt := &fakeSTT{err: healthErr}
	controller := NewController(telegram, stt, &fakeConfig{})

	err := controller.HandleCommand(context.Background(), adminChatID, "/status")
	if !errors.Is(err, ErrSTTHealth) {
		t.Fatalf("HandleCommand() error = %v, want errors.Is ErrSTTHealth", err)
	}

	if !errors.Is(err, healthErr) {
		t.Errorf("HandleCommand() error = %v, want it to wrap the provider error", err)
	}

	if len(telegram.messages) != 1 || telegram.messages[0].text != msgSTTUnhealthy {
		t.Errorf("messages = %+v, want single %q message", telegram.messages, msgSTTUnhealthy)
	}
}

func TestHandleCommand_StatusSendErrorIsJoined(t *testing.T) {
	healthErr := errors.New("connection refused")
	sendErr := errors.New("chat not found")
	telegram := &fakeTelegram{sendErr: sendErr}
	stt := &fakeSTT{err: healthErr}
	controller := NewController(telegram, stt, &fakeConfig{})

	err := controller.HandleCommand(context.Background(), adminChatID, "/status")
	if !errors.Is(err, ErrSTTHealth) {
		t.Errorf("HandleCommand() error = %v, want errors.Is ErrSTTHealth", err)
	}

	if !errors.Is(err, ErrSendMessage) {
		t.Errorf("HandleCommand() error = %v, want errors.Is ErrSendMessage", err)
	}

	if !errors.Is(err, sendErr) {
		t.Errorf("HandleCommand() error = %v, want it to wrap the provider error", err)
	}
}

func TestHandleCommand_Chats(t *testing.T) {
	telegram := &fakeTelegram{}
	stt := &fakeSTT{health: "unused"}
	controller := NewController(telegram, stt, &fakeConfig{allowed: []int64{-100123, 42}})

	if err := controller.HandleCommand(context.Background(), adminChatID, "/chats"); err != nil {
		t.Fatalf("HandleCommand() unexpected error: %v", err)
	}

	if stt.calls != 0 {
		t.Errorf("Health called %d times, want 0 for /chats", stt.calls)
	}

	want := sentMessage{chatID: adminChatID, text: chatsHeader + "\n-100123\n42"}
	if len(telegram.messages) != 1 || telegram.messages[0] != want {
		t.Errorf("messages = %+v, want [%+v]", telegram.messages, want)
	}
}

func TestHandleCommand_ChatsEmptyWhitelist(t *testing.T) {
	telegram := &fakeTelegram{}
	controller := NewController(telegram, &fakeSTT{}, &fakeConfig{})

	if err := controller.HandleCommand(context.Background(), adminChatID, "/chats"); err != nil {
		t.Fatalf("HandleCommand() unexpected error: %v", err)
	}

	if len(telegram.messages) != 1 || telegram.messages[0].text != msgNoChats {
		t.Errorf("messages = %+v, want single %q message", telegram.messages, msgNoChats)
	}
}

func TestHandleCommand_ChatsSendError(t *testing.T) {
	sendErr := errors.New("chat not found")
	telegram := &fakeTelegram{sendErr: sendErr}
	controller := NewController(telegram, &fakeSTT{}, &fakeConfig{allowed: []int64{7}})

	err := controller.HandleCommand(context.Background(), adminChatID, "/chats")
	if !errors.Is(err, ErrSendMessage) {
		t.Fatalf("HandleCommand() error = %v, want errors.Is ErrSendMessage", err)
	}

	if !errors.Is(err, sendErr) {
		t.Errorf("HandleCommand() error = %v, want it to wrap the provider error", err)
	}
}

func TestHandleCommand_UnknownCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{name: "unknown command", command: "/restart"},
		{name: "plain text", command: "привет"},
		{name: "empty", command: ""},
		{name: "whitespace only", command: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telegram := &fakeTelegram{}
			stt := &fakeSTT{health: "unused"}
			controller := NewController(telegram, stt, &fakeConfig{allowed: []int64{7}})

			if err := controller.HandleCommand(context.Background(), adminChatID, tt.command); err != nil {
				t.Fatalf("HandleCommand() unexpected error: %v", err)
			}

			if stt.calls != 0 {
				t.Errorf("Health called %d times, want 0 for an unknown command", stt.calls)
			}

			if len(telegram.messages) != 1 || telegram.messages[0].text != msgUnknownCommand {
				t.Errorf("messages = %+v, want single hint message", telegram.messages)
			}
		})
	}
}

func TestHandleCommand_NormalizesCommand(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		wantHealth bool
	}{
		{name: "bot mention", command: "/status@tg_stt_bot", wantHealth: true},
		{name: "trailing argument", command: "/status now", wantHealth: true},
		{name: "uppercase", command: "/STATUS", wantHealth: true},
		{name: "surrounding spaces", command: "  /status  ", wantHealth: true},
		{name: "no slash", command: "status", wantHealth: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telegram := &fakeTelegram{}
			stt := &fakeSTT{health: `{"status":"ok"}`}
			controller := NewController(telegram, stt, &fakeConfig{})

			if err := controller.HandleCommand(context.Background(), adminChatID, tt.command); err != nil {
				t.Fatalf("HandleCommand() unexpected error: %v", err)
			}

			gotHealth := stt.calls == 1
			if gotHealth != tt.wantHealth {
				t.Fatalf("Health called = %v, want %v", gotHealth, tt.wantHealth)
			}

			if len(telegram.messages) != 1 {
				t.Fatalf("messages = %+v, want exactly one", telegram.messages)
			}

			if tt.wantHealth && !strings.HasPrefix(telegram.messages[0].text, statusPrefix) {
				t.Errorf("message = %q, want the status report", telegram.messages[0].text)
			}
		})
	}
}
