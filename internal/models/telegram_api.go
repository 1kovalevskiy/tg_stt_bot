package models

// Telegram Bot API protocol constants. They describe the API itself, not the
// bot's settings, so they are compile-time values rather than config options.
const (
	// TelegramFileBaseURL is the Bot API host used to download files by file_path.
	TelegramFileBaseURL = "https://api.telegram.org"
	// TelegramMessageLimit is the maximum Telegram message length in UTF-16 code units.
	TelegramMessageLimit = 4096
	// TelegramMaxFileSize is the Bot API download limit: getFile refuses bigger
	// files, so they are rejected before any download attempt. The STT provider
	// bounds its buffered request body by the same value: a longer stream means
	// the caller skipped its own check.
	TelegramMaxFileSize = 20 << 20 // 20 MB
)
