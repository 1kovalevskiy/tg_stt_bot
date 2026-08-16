package adminController

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/1kovalevskiy/tg_stt_bot/internal/controllers"
	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

// sentMessage records a single SendMessage call.
type sentMessage struct {
	chatID int64
	text   string
}

// fakeTelegram is a hand-written fake of the telegramProvider interface.
type fakeTelegram struct {
	messages []sentMessage
	gotCtx   context.Context
	sendErr  error
}

func (f *fakeTelegram) SendMessage(ctx context.Context, chatID int64, text string) error {
	f.messages = append(f.messages, sentMessage{chatID: chatID, text: text})
	f.gotCtx = ctx

	if f.sendErr != nil {
		return f.sendErr
	}

	return nil
}

// fakeSTT is a hand-written fake of the sttProvider interface.
type fakeSTT struct {
	calls  int
	gotCtx context.Context
	health string
	err    error
}

func (f *fakeSTT) Health(ctx context.Context) (string, error) {
	f.calls++
	f.gotCtx = ctx

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
	if !errors.Is(err, controllers.ErrSTTHealth) {
		t.Fatalf("HandleCommand() error = %v, want errors.Is controllers.ErrSTTHealth", err)
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
	if !errors.Is(err, controllers.ErrSTTHealth) {
		t.Errorf("HandleCommand() error = %v, want errors.Is controllers.ErrSTTHealth", err)
	}

	if !errors.Is(err, controllers.ErrSendMessage) {
		t.Errorf("HandleCommand() error = %v, want errors.Is controllers.ErrSendMessage", err)
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
	if !errors.Is(err, controllers.ErrSendMessage) {
		t.Fatalf("HandleCommand() error = %v, want errors.Is controllers.ErrSendMessage", err)
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

func TestHandleCommand_NormalizesChatsCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		wantChats bool
	}{
		{name: "bot mention", command: "/chats@tg_stt_bot", wantChats: true},
		{name: "uppercase", command: "/CHATS", wantChats: true},
		{name: "trailing argument", command: "/chats -100500", wantChats: true},
		{name: "unknown command", command: "/chatz", wantChats: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telegram := &fakeTelegram{}
			controller := NewController(telegram, &fakeSTT{}, &fakeConfig{allowed: []int64{-100500}})

			if err := controller.HandleCommand(context.Background(), adminChatID, tt.command); err != nil {
				t.Fatalf("HandleCommand() unexpected error: %v", err)
			}

			if len(telegram.messages) != 1 {
				t.Fatalf("messages = %+v, want exactly one", telegram.messages)
			}

			gotChats := strings.HasPrefix(telegram.messages[0].text, chatsHeader)
			if gotChats != tt.wantChats {
				t.Errorf("whitelist reported = %v, want %v for %q", gotChats, tt.wantChats, tt.command)
			}
		})
	}
}

func TestHandleCommand_LongAnswersAreSplit(t *testing.T) {
	// A response over the Telegram limit is rejected as a whole, so both
	// answers have to be chunked.
	allowed := make([]int64, 0, 500)
	for i := 0; i < 500; i++ {
		allowed = append(allowed, int64(-1000000000000-i))
	}

	tests := []struct {
		name    string
		command string
		health  string
		config  *fakeConfig
	}{
		{
			name:    "long whitelist",
			command: "/chats",
			config:  &fakeConfig{allowed: allowed},
		},
		{
			// A misrouted base_url can answer 200 with a whole HTML page.
			name:    "long health response",
			command: "/status",
			health:  strings.Repeat("a", 3*models.TelegramMessageLimit),
			config:  &fakeConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			telegram := &fakeTelegram{}
			controller := NewController(telegram, &fakeSTT{health: tt.health}, tt.config)

			if err := controller.HandleCommand(context.Background(), adminChatID, tt.command); err != nil {
				t.Fatalf("HandleCommand() unexpected error: %v", err)
			}

			if len(telegram.messages) < 2 {
				t.Fatalf("messages = %d, want the answer split into several", len(telegram.messages))
			}

			for i, message := range telegram.messages {
				if chunks := models.SplitText(message.text, models.TelegramMessageLimit); len(chunks) != 1 {
					t.Errorf("message %d does not fit into a single Telegram message", i)
				}
			}
		})
	}
}

func TestHandleCommand_ChatsListsEveryAllowedChat(t *testing.T) {
	telegram := &fakeTelegram{}
	allowed := []int64{-100500, 42}
	controller := NewController(telegram, &fakeSTT{}, &fakeConfig{allowed: allowed})

	if err := controller.HandleCommand(context.Background(), adminChatID, "/chats"); err != nil {
		t.Fatalf("HandleCommand() unexpected error: %v", err)
	}

	joined := ""
	for _, message := range telegram.messages {
		joined += message.text + "\n"
	}

	for _, id := range allowed {
		if !strings.Contains(joined, strconv.FormatInt(id, 10)) {
			t.Errorf("answer %q does not list chat %d", joined, id)
		}
	}
}

func TestHandleCommand_StatusProbeHasItsOwnTimeout(t *testing.T) {
	telegram := &fakeTelegram{}
	stt := &fakeSTT{health: `{"status":"ok"}`}
	controller := NewController(telegram, stt, &fakeConfig{})

	// A transcription-sized budget on the caller's side: a health probe must
	// not make the admin wait that long for an answer.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := controller.HandleCommand(ctx, adminChatID, "/status"); err != nil {
		t.Fatalf("HandleCommand() unexpected error: %v", err)
	}

	deadline, ok := stt.gotCtx.Deadline()
	if !ok {
		t.Fatal("health probe context has no deadline, want the health timeout applied")
	}

	if left := time.Until(deadline); left > healthTimeout {
		t.Errorf("health probe has %v left, want at most %v", left, healthTimeout)
	}
}

func TestHandleCommand_StatusReplyDoesNotInheritTheProbeDeadline(t *testing.T) {
	telegram := &fakeTelegram{}
	stt := &fakeSTT{err: errors.New("connection refused")}
	controller := NewController(telegram, stt, &fakeConfig{})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := controller.HandleCommand(ctx, adminChatID, "/status"); !errors.Is(err, controllers.ErrSTTHealth) {
		t.Fatalf("HandleCommand() error = %v, want errors.Is controllers.ErrSTTHealth", err)
	}

	// The probe deadline must be gone by the time the answer is sent, or a
	// hung service would also swallow the "unavailable" notice.
	deadline, ok := telegram.gotCtx.Deadline()
	if !ok {
		t.Fatal("send context has no deadline, want the caller's one")
	}

	if left := time.Until(deadline); left <= healthTimeout {
		t.Errorf("send context has %v left, want more than the health timeout %v", left, healthTimeout)
	}
}
