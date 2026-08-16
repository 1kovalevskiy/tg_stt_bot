package models

import "strings"

// RedactedToken is what replaces the bot token in text shown to a human.
const RedactedToken = "[REDACTED]"

// RedactToken removes every occurrence of the bot token from text: transport
// and library errors embed the request URL, which contains the token. An empty
// token leaves the text untouched, so a misconfigured bot cannot accidentally
// redact everything.
func RedactToken(text, token string) string {
	if token == "" {
		return text
	}

	return strings.ReplaceAll(text, token, RedactedToken)
}
