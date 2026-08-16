package models

import "testing"

func TestIsChatAllowed(t *testing.T) {
	t.Parallel()

	const adminID = int64(100)

	allowed := []int64{-1001234567890, 42}

	tests := []struct {
		name    string
		chatID  int64
		allowed []int64
		want    bool
	}{
		{
			name:    "group chat from whitelist",
			chatID:  -1001234567890,
			allowed: allowed,
			want:    true,
		},
		{
			name:    "private chat from whitelist",
			chatID:  42,
			allowed: allowed,
			want:    true,
		},
		{
			name:    "foreign group chat",
			chatID:  -1009999999999,
			allowed: allowed,
			want:    false,
		},
		{
			name:    "admin private chat",
			chatID:  adminID,
			allowed: allowed,
			want:    true,
		},
		{
			name:    "admin private chat with empty whitelist",
			chatID:  adminID,
			allowed: nil,
			want:    true,
		},
		{
			name:    "foreign private chat",
			chatID:  777,
			allowed: allowed,
			want:    false,
		},
		{
			name:    "foreign chat with empty whitelist",
			chatID:  -500,
			allowed: nil,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsChatAllowed(tt.chatID, adminID, tt.allowed); got != tt.want {
				t.Errorf("IsChatAllowed(%d, %d, %v) = %v, want %v",
					tt.chatID, adminID, tt.allowed, got, tt.want)
			}
		})
	}
}
