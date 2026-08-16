package models

import "strings"

// ParseCommandName extracts the command name from a message: arguments are
// dropped, the "@botname" suffix Telegram adds in groups is stripped and the
// name is lowercased. The transport logs the result and the admin controller
// routes on it, so both have to read the same word out of the same text.
func ParseCommandName(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}

	name, _, _ := strings.Cut(fields[0], "@")

	return strings.ToLower(name)
}
