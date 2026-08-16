package models

// Bot commands available in the admin's private chat.
const (
	// CommandPrefix marks a message as a bot command.
	CommandPrefix = "/"
	// CommandStatus checks the STT service.
	CommandStatus = CommandPrefix + "status"
	// CommandChats prints the configured chat whitelist.
	CommandChats = CommandPrefix + "chats"
)
