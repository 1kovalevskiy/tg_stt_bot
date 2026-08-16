package telegram

import (
	"context"
	"testing"

	tgmodels "github.com/go-telegram/bot/models"
)

func TestDispatcher_AdminCommand(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	tests := []struct {
		name   string
		update *tgmodels.Update
	}{
		{name: "allowed group chat", update: textUpdate(testAllowedID, "/status")},
		{name: "foreign chat", update: textUpdate(testForeignID, "/status")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Own fakes per subtest: a shared dispatcher would accumulate
			// calls and make the assertions depend on the subtest order.
			d, _, admin := newTestDispatcher()

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
	t.Parallel()

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
			t.Parallel()

			d, _, admin := newTestDispatcher()
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
	t.Parallel()

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
