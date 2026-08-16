package models

import (
	"strings"
	"testing"
)

// TestCommandsCarryThePrefix pins that every command matches what the update
// dispatcher looks for: it routes a message to the admin controller only when
// the text starts with CommandPrefix, so a command without it is unreachable.
func TestCommandsCarryThePrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
	}{
		{name: "status", command: CommandStatus},
		{name: "chats", command: CommandChats},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if !strings.HasPrefix(tt.command, CommandPrefix) {
				t.Errorf("command %q does not start with %q", tt.command, CommandPrefix)
			}

			if tt.command != strings.ToLower(tt.command) {
				// normalizeCommand lowercases the incoming command before the
				// comparison, so an uppercase constant would never match.
				t.Errorf("command %q is not lowercase", tt.command)
			}
		})
	}
}
