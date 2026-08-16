package models

import "time"

// Application runtime.
const (
	// DefaultConfigPath is used when no -config flag is given.
	DefaultConfigPath = "config.json"
	// BotWorkers is the number of concurrent update handlers: transcription
	// takes seconds and the library default of one worker would make every
	// chat wait for the previous one. Together with WithNotAsyncHandlers the
	// number is also the concurrency bound — without it the library runs a
	// goroutine per update and a burst of voice messages would hold an
	// unbounded number of downloaded files in memory.
	BotWorkers = 4
)

// Telegram Bot API protocol. These describe the API itself, not the bot's
// settings, so they are compile-time values rather than config options.
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

// parakeet STT API: endpoints, multipart field names and response limits.
const (
	// STTTranscriptionsPath is the OpenAI-compatible transcription endpoint.
	STTTranscriptionsPath = "/v1/audio/transcriptions"
	// STTHealthPath is the parakeet health endpoint.
	STTHealthPath = "/health"
	// STTFileField is the multipart field carrying the audio file.
	STTFileField = "file"
	// STTLanguageField is the multipart field carrying the recognition language.
	STTLanguageField = "language"
	// STTMaxResponseSize limits how much of a service response is read into memory.
	STTMaxResponseSize = 1 << 20 // 1 MB
	// STTMaxErrorSnippet limits how much of a non-JSON error body ends up in an error.
	STTMaxErrorSnippet = 256
)

// File names reported to the STT service for each kind of incoming audio.
// parakeet detects the container by content, so the extension is
// informational only.
const (
	// VoiceFilename is the file name used for voice messages.
	VoiceFilename = "voice.oga"
	// VideoNoteFilename is the file name used for video notes (round videos).
	VideoNoteFilename = "video_note.mp4"
)

// Bot commands available in the admin's private chat.
const (
	// CommandPrefix marks a message as a bot command.
	CommandPrefix = "/"
	// CommandStatus checks the STT service.
	CommandStatus = CommandPrefix + "status"
	// CommandChats prints the configured chat whitelist.
	CommandChats = CommandPrefix + "chats"
)

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

// Timeouts controllers derive on their own. They bound protocol-level steps
// rather than a whole external call, so they are constants and not settings.
const (
	// FailureReplyTimeout bounds the failure notice, which is sent with a
	// context detached from the caller's cancellation and would otherwise
	// hold up the shutdown for a full Bot API timeout.
	FailureReplyTimeout = 5 * time.Second
	// HealthProbeTimeout bounds the /status probe. Without it the probe
	// inherits the full transcription budget, and a hung service would keep
	// the admin waiting minutes for an answer that is supposed to be immediate.
	HealthProbeTimeout = 10 * time.Second
)

// Service chat log delivery: queue size, delivery timeouts and the rate limit
// that keeps a repeating failure from flooding the chat.
const (
	// ServiceChatQueueSize is how many ERROR records may wait for delivery
	// before new ones are dropped instead of blocking the logging caller.
	ServiceChatQueueSize = 64
	// ServiceChatSendTimeout bounds a single delivery attempt.
	ServiceChatSendTimeout = 15 * time.Second
	// ServiceChatDrainTimeout bounds the queue drain on shutdown. It has to
	// exceed a single send timeout, otherwise Close reports a failure for a
	// delivery that is still perfectly on time.
	ServiceChatDrainTimeout = ServiceChatSendTimeout + 5*time.Second
	// ServiceChatRateWindow and ServiceChatRateBurst cap how many records may
	// be delivered per window: a repeating failure would otherwise turn into
	// one Telegram message per occurrence, flooding the chat and burning the
	// send quota shared with user replies.
	ServiceChatRateWindow = time.Minute
	ServiceChatRateBurst  = 10
	// MsgSuppressedFormat reports how many records the rate limit dropped.
	MsgSuppressedFormat = "%d error records suppressed by the service chat rate limit"
)
