package models

import (
	"strings"
	"testing"
)

func TestRedactToken(t *testing.T) {
	t.Parallel()

	const token = "123456:AAHfake-token"

	tests := []struct {
		name  string
		text  string
		token string
		want  string
	}{
		{
			name:  "token in url is redacted",
			text:  `Get "https://api.telegram.org/bot123456:AAHfake-token/getMe": timeout`,
			token: token,
			want:  `Get "https://api.telegram.org/bot[REDACTED]/getMe": timeout`,
		},
		{
			name:  "several occurrences",
			text:  token + " and " + token,
			token: token,
			want:  "[REDACTED] and [REDACTED]",
		},
		{
			name:  "no token in text",
			text:  "error call getMe, unauthorized",
			token: token,
			want:  "error call getMe, unauthorized",
		},
		{
			name:  "empty token leaves text intact",
			text:  "some error",
			token: "",
			want:  "some error",
		},
		{
			name:  "token is the whole text",
			text:  token,
			token: token,
			want:  RedactedToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := RedactToken(tt.text, tt.token)
			if got != tt.want {
				t.Errorf("RedactToken() = %q, want %q", got, tt.want)
			}

			if tt.token != "" && strings.Contains(got, tt.token) {
				t.Errorf("RedactToken() leaked the token: %q", got)
			}
		})
	}
}
