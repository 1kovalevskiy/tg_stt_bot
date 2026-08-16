package models

import "testing"

func TestParseCommandName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "plain command", text: CommandStatus, want: CommandStatus},
		{name: "command with argument", text: "/status arg", want: CommandStatus},
		{name: "command with several arguments", text: "/chats -100500 42", want: CommandChats},
		{name: "command addressed to the bot", text: "/chats@botname", want: CommandChats},
		{name: "command addressed to the bot with an argument", text: "/status@botname now", want: CommandStatus},
		{name: "uppercase command", text: "/CHATS", want: CommandChats},
		{name: "mixed case command addressed to the bot", text: "/StAtUs@BotName", want: CommandStatus},
		{name: "padded command", text: "  /chats  ", want: CommandChats},
		{name: "empty input", text: "", want: ""},
		{name: "blank input", text: "   \t\n ", want: ""},
		{name: "plain text is returned as its first word", text: "привет мир", want: "привет"},
		{name: "text without a command prefix", text: "status", want: "status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ParseCommandName(tt.text); got != tt.want {
				t.Errorf("ParseCommandName(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}
