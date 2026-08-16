package models

import "slices"

// IsChatAllowed reports whether the bot should process messages from chatID:
// either the chat is in the whitelist or it is the admin's private chat
// (in Telegram the private chat ID equals the user ID).
func IsChatAllowed(chatID, adminID int64, allowed []int64) bool {
	if chatID == adminID {
		return true
	}

	return slices.Contains(allowed, chatID)
}
