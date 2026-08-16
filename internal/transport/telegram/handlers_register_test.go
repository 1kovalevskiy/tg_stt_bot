package telegram

import (
	"testing"

	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// registration is a single matcher/handler pair passed to the bot client.
type registration struct {
	match   bot.MatchFunc
	handler bot.HandlerFunc
}

// fakeRegistrar is a hand-written fake of the botRegistrar interface.
type fakeRegistrar struct {
	registrations []registration
}

func (f *fakeRegistrar) RegisterHandlerMatchFunc(
	matchFunc bot.MatchFunc, handler bot.HandlerFunc, _ ...bot.Middleware,
) string {
	f.registrations = append(f.registrations, registration{match: matchFunc, handler: handler})

	return "handler-id"
}

// TestRegisterHandlers_MatchersArePairedWithTheirHandlers pins the pairing
// itself: a matcher registered with the wrong handler would silently drop
// every update it matches, and no dispatcher-level test would notice.
func TestRegisterHandlers_MatchersArePairedWithTheirHandlers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		update *tgmodels.Update
		want   func(*fakeChatController, *fakeAdminController) bool
	}{
		{
			name:   "voice",
			update: voiceUpdate(testAllowedID, 1),
			want: func(chat *fakeChatController, _ *fakeAdminController) bool {
				return len(chat.voice) == 1
			},
		},
		{
			name:   "video note",
			update: videoNoteUpdate(testAllowedID, 2),
			want: func(chat *fakeChatController, _ *fakeAdminController) bool {
				return len(chat.videoNote) == 1
			},
		},
		{
			name:   "admin command",
			update: textUpdate(testAdminID, "/status"),
			want: func(_ *fakeChatController, admin *fakeAdminController) bool {
				return len(admin.calls) == 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d, chat, admin := newTestDispatcher()
			registrar := &fakeRegistrar{}

			d.RegisterHandlers(registrar)

			if len(registrar.registrations) != 3 {
				t.Fatalf("registered %d handlers, want 3", len(registrar.registrations))
			}

			matched := 0

			for _, reg := range registrar.registrations {
				if !reg.match(tt.update) {
					continue
				}

				matched++

				reg.handler(t.Context(), nil, tt.update)
			}

			if matched != 1 {
				t.Fatalf("%d matchers accepted the update, want exactly 1", matched)
			}

			if !tt.want(chat, admin) {
				t.Error("the matched handler did not call the controller the matcher stands for")
			}
		})
	}
}
