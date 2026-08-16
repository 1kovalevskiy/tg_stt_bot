package configs

import "errors"

// Reading the config file and every rejection validateConfig can report. A
// config that does not pass validation stops the bot at its first init step,
// so each sentinel names the exact setting the operator has to fix.
var (
	// ErrNilConfig reports validation called without a config at all.
	ErrNilConfig = errors.New("config is nil")
	// ErrReadConfig reports a config file that could not be read or parsed.
	ErrReadConfig = errors.New("failed to read config file")
	// ErrEmptyTelegramToken reports a missing bot token: nothing can be sent without it.
	ErrEmptyTelegramToken = errors.New("telegram.token is empty")
	// ErrZeroTelegramAdminID reports a missing admin chat: the commands would have no owner.
	ErrZeroTelegramAdminID = errors.New("telegram.admin_id is zero")
	// ErrZeroTelegramServiceChatID reports a missing service chat: ERROR records would go nowhere.
	ErrZeroTelegramServiceChatID = errors.New("telegram.service_chat_id is zero")
	// ErrZeroAllowedChat reports a zero id in the whitelist, which matches no chat.
	ErrZeroAllowedChat = errors.New("telegram.allowed_chats contains zero element")
	// ErrInvalidTelegramAPITimeout reports an unparseable or non-positive Bot API timeout.
	ErrInvalidTelegramAPITimeout = errors.New("telegram.api_timeout is invalid")
	// ErrInvalidTelegramDownloadTimeout reports an unparseable or non-positive download timeout.
	ErrInvalidTelegramDownloadTimeout = errors.New("telegram.download_timeout is invalid")
	// ErrInvalidSTTBaseURL reports an STT address without a scheme or a host.
	ErrInvalidSTTBaseURL = errors.New("stt.base_url is invalid")
	// ErrInvalidSTTTimeout reports an unparseable or non-positive STT timeout.
	ErrInvalidSTTTimeout = errors.New("stt.timeout is invalid")
)
