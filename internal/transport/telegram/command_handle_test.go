package telegram

import "testing"

func TestParseCommandName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "plain command", text: "/status", want: "/status"},
		{name: "command with argument", text: "/chats -100500", want: "/chats"},
		{name: "command addressed to the bot", text: "/status@my_bot", want: "/status"},
		{name: "padded command", text: "  /chats  ", want: "/chats"},
		{name: "empty text", text: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := parseCommandName(tt.text); got != tt.want {
				t.Errorf("parseCommandName(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}
