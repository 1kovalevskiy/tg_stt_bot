package app

import "errors"

// Wiring errors only: a step that cannot run because an earlier one did not
// produce its dependency, and the two failures the wiring owns itself
// (reading the config file and closing the registered resources). Everything
// a layer can fail at is that layer's own sentinel.
var (
	ErrNilConfig           = errors.New("app config is nil")
	ErrNilLogHandler       = errors.New("base log handler is not initialized")
	ErrNilTelegramProvider = errors.New("telegram provider is not initialized")
	ErrNilSTTProvider      = errors.New("stt provider is not initialized")
	ErrNilController       = errors.New("controller is not initialized")
	ErrNilBot              = errors.New("telegram bot client is not initialized")
	ErrReadConfig          = errors.New("failed to read config")
	ErrCloseResources      = errors.New("failed to close app resources")
)
