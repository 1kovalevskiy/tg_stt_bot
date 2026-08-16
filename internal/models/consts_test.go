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
				// ParseCommandName lowercases the incoming command before the
				// comparison, so an uppercase constant would never match.
				t.Errorf("command %q is not lowercase", tt.command)
			}
		})
	}
}

// TestServiceChatDrainTimeoutExceedsSendTimeout pins the relation the sink
// relies on: the drain has to outlast a single delivery attempt, otherwise
// Close reports a failure for a send that is still perfectly on time.
func TestServiceChatDrainTimeoutExceedsSendTimeout(t *testing.T) {
	t.Parallel()

	if ServiceChatDrainTimeout <= ServiceChatSendTimeout {
		t.Errorf("ServiceChatDrainTimeout = %v, want more than ServiceChatSendTimeout %v",
			ServiceChatDrainTimeout, ServiceChatSendTimeout)
	}
}

// TestServiceChatRateLimitIsPositive pins that the rate limit lets records
// through at all: a zero burst would silence the service chat completely.
func TestServiceChatRateLimitIsPositive(t *testing.T) {
	t.Parallel()

	if ServiceChatRateBurst <= 0 {
		t.Errorf("ServiceChatRateBurst = %d, want a positive burst", ServiceChatRateBurst)
	}

	if ServiceChatRateWindow <= 0 {
		t.Errorf("ServiceChatRateWindow = %v, want a positive window", ServiceChatRateWindow)
	}
}
