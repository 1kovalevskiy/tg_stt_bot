package models

// IncomingAudio is a pure model of an incoming voice or video note message.
// MessageID is int and ChatID/FileSize are int64 to match the Telegram bot library types.
type IncomingAudio struct {
	ChatID    int64
	MessageID int
	FileID    string
	FileSize  int64
}
