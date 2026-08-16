package app

import "errors"

// Wiring errors only: a step that cannot run because an earlier one did not
// produce its dependency, and the two failures the wiring owns itself
// (reading the config file and closing the registered resources). Everything
// a layer can fail at is that layer's own sentinel.
var (
	// ErrNilConfig reports a step running before the config was read.
	ErrNilConfig = errors.New("app config is nil")
	// ErrNilLogHandler reports a step running before the base log handler was built.
	ErrNilLogHandler = errors.New("base log handler is not initialized")
	// ErrNilTelegramProvider reports a step running before the telegram provider was built.
	ErrNilTelegramProvider = errors.New("telegram provider is not initialized")
	// ErrNilSTTProvider reports a step running before the stt provider was built.
	ErrNilSTTProvider = errors.New("stt provider is not initialized")
	// ErrNilController reports handler registration running before the controllers were built.
	ErrNilController = errors.New("controller is not initialized")
	// ErrNilBot reports a step running before the bot client was created.
	ErrNilBot = errors.New("telegram bot client is not initialized")
	// ErrReadConfig reports a config file the wiring could not read.
	ErrReadConfig = errors.New("failed to read config")
	// ErrCloseResources reports at least one shutdown hook failing; it wraps all of them.
	ErrCloseResources = errors.New("failed to close app resources")
)
