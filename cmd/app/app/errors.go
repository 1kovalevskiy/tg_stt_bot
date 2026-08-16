package app

import "errors"

var (
	ErrNilConfig           = errors.New("app config is nil")
	ErrNilLogHandler       = errors.New("base log handler is not initialized")
	ErrNilTelegramProvider = errors.New("telegram provider is not initialized")
	ErrNilSTTProvider      = errors.New("stt provider is not initialized")
	ErrNilController       = errors.New("controller is not initialized")
	ErrNilBot              = errors.New("telegram bot client is not initialized")
	ErrReadConfig          = errors.New("failed to read config")
	ErrCreateBot           = errors.New("failed to create telegram bot client")
	ErrCloseResources      = errors.New("failed to close app resources")
	ErrLogSinkDrainTimeout = errors.New("service chat log sink drain timed out")
)
