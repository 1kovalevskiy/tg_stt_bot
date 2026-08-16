package config

import "errors"

var (
	ErrNilConfig                 = errors.New("config is nil")
	ErrReadConfig                = errors.New("failed to read config file")
	ErrReadEnv                   = errors.New("failed to read environment")
	ErrEmptyTelegramToken        = errors.New("telegram.token is empty")
	ErrZeroTelegramAdminID       = errors.New("telegram.admin_id is zero")
	ErrZeroTelegramServiceChatID = errors.New("telegram.service_chat_id is zero")
	ErrZeroAllowedChat           = errors.New("telegram.allowed_chats contains zero element")
	ErrInvalidSTTBaseURL         = errors.New("stt.base_url is invalid")
	ErrInvalidSTTTimeout         = errors.New("stt.timeout is invalid")
)
