package models

// User-facing texts the bot sends to a chat.
const (
	// MsgFileTooLarge reports audio the Bot API refuses to serve.
	MsgFileTooLarge = "Файл слишком большой, я не могу его скачать."
	// MsgNoSpeech reports an empty transcription.
	MsgNoSpeech = "Речь не распознана."
	// MsgTranscribeFailed reports a failed transcription.
	MsgTranscribeFailed = "Не удалось расшифровать сообщение."
	// MsgSTTUnhealthy reports an unreachable STT service.
	MsgSTTUnhealthy = "parakeet недоступен."
	// MsgNoChats reports an empty chat whitelist.
	MsgNoChats = "Список разрешённых чатов пуст."
	// MsgUnknownCommand hints at the commands the admin can run.
	MsgUnknownCommand = "Неизвестная команда. Доступны: /status — проверка parakeet, /chats — список разрешённых чатов."
	// StatusPrefix precedes the raw STT health response in the /status answer.
	StatusPrefix = "parakeet: "
	// ChatsHeader precedes the chat whitelist in the /chats answer.
	ChatsHeader = "Разрешённые чаты:"
)
