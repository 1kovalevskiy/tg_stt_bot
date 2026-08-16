package models

// Application runtime constants shared by the entry point and the wiring.
const (
	// DefaultConfigPath is used when no -config flag is given.
	DefaultConfigPath = "config.json"
	// BotWorkers is the number of concurrent update handlers. Transcription
	// takes seconds, and the library default of one worker would make every
	// chat wait for the previous one.
	//
	// The number only bounds concurrency together with WithNotAsyncHandlers:
	// by default the library runs every handler in its own goroutine and the
	// workers just drain the updates channel, so a burst of voice messages
	// would hold an unbounded number of downloaded files in memory.
	BotWorkers = 4
)
