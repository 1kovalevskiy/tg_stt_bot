// Package configs reads the bot settings from a JSON file with the environment
// applied on top of it, validates them once at startup and hands them out
// through getters. Consumers never touch the struct fields: they take the
// config as an interface with the getters they need.
package configs

import (
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Config is the validated application config. It is built by NewConfig alone
// and read through its getters.
type Config struct {
	App      App      `json:"app"`
	Telegram Telegram `json:"telegram"`
	STT      STT      `json:"stt"`

	// Timeouts are configured as strings and parsed once, after validation
	// has proven every one of them parseable and positive. cleanenv skips
	// unexported fields, so these are filled in by NewConfig alone.
	telegramAPITimeout      time.Duration
	telegramDownloadTimeout time.Duration
	sttTimeout              time.Duration
}

// App holds the settings of the application itself.
type App struct {
	LogLevel string `json:"log_level" env:"APP_LOG_LEVEL" env-default:"INFO"`
}

// Telegram holds the bot credentials, the chats it serves and its Bot API timeouts.
type Telegram struct {
	Token           string  `json:"token" env:"TELEGRAM_TOKEN"`
	AdminID         int64   `json:"admin_id" env:"TELEGRAM_ADMIN_ID"`
	ServiceChatID   int64   `json:"service_chat_id" env:"TELEGRAM_SERVICE_CHAT_ID"`
	AllowedChats    []int64 `json:"allowed_chats" env:"TELEGRAM_ALLOWED_CHATS"`
	APITimeout      string  `json:"api_timeout" env:"TELEGRAM_API_TIMEOUT" env-default:"30s"`
	DownloadTimeout string  `json:"download_timeout" env:"TELEGRAM_DOWNLOAD_TIMEOUT" env-default:"2m"`
}

// STT holds the address, recognition language and timeout of the STT service.
type STT struct {
	BaseURL  string `json:"base_url" env:"STT_BASE_URL"`
	Language string `json:"language" env:"STT_LANGUAGE"`
	Timeout  string `json:"timeout" env:"STT_TIMEOUT" env-default:"120s"`
}

// init loads the .env file if there is one; a missing file is not an error.
func init() {
	if err := godotenv.Load(); err != nil {
		slog.Debug("no .env file found")
	}
}

// NewConfig reads the config file, applies the environment on top of it
// (cleanenv.ReadConfig does both) and validates the result.
func NewConfig(path string) (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadConfig, err)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// GetAppLogLevel returns the configured slog level name, upper-cased.
func (c Config) GetAppLogLevel() string {
	return strings.ToUpper(strings.TrimSpace(c.App.LogLevel))
}

// GetTelegramToken returns the bot token. It is read only where the client is
// created and never lands in a log or an error text.
func (c Config) GetTelegramToken() string {
	return strings.TrimSpace(c.Telegram.Token)
}

// GetTelegramAdminID returns the chat id of the admin's private chat, the only
// place the bot takes commands from.
func (c Config) GetTelegramAdminID() int64 {
	return c.Telegram.AdminID
}

// GetTelegramServiceChatID returns the chat the ERROR records are mirrored to.
func (c Config) GetTelegramServiceChatID() int64 {
	return c.Telegram.ServiceChatID
}

// GetTelegramAllowedChats returns the chat whitelist: audio from anywhere else
// is ignored.
func (c Config) GetTelegramAllowedChats() []int64 {
	return c.Telegram.AllowedChats
}

// GetTelegramAPITimeout bounds a single Bot API call (getFile, sendMessage).
func (c Config) GetTelegramAPITimeout() time.Duration {
	return c.telegramAPITimeout
}

// GetTelegramDownloadTimeout bounds a single file download, including the
// time the caller spends reading the body.
func (c Config) GetTelegramDownloadTimeout() time.Duration {
	return c.telegramDownloadTimeout
}

// GetSTTBaseURL returns the STT service address without its trailing slash.
func (c Config) GetSTTBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(c.STT.BaseURL), "/")
}

// GetSTTLanguage returns the recognition language. An empty one leaves the
// multipart field out and lets the service detect the language itself.
func (c Config) GetSTTLanguage() string {
	return strings.TrimSpace(c.STT.Language)
}

// GetSTTTimeout bounds a single call to the STT service.
func (c Config) GetSTTTimeout() time.Duration {
	return c.sttTimeout
}

// validateConfig rejects a config the bot cannot run with and caches the
// parsed timeouts it has just proven correct, so the getters never re-parse
// a string and never have to invent a fallback for an unparseable one.
func validateConfig(cfg *Config) error {
	if cfg == nil {
		return ErrNilConfig
	}

	if strings.TrimSpace(cfg.Telegram.Token) == "" {
		return ErrEmptyTelegramToken
	}

	if cfg.Telegram.AdminID == 0 {
		return ErrZeroTelegramAdminID
	}

	if cfg.Telegram.ServiceChatID == 0 {
		return ErrZeroTelegramServiceChatID
	}

	if slices.Contains(cfg.Telegram.AllowedChats, 0) {
		return ErrZeroAllowedChat
	}

	apiTimeout, err := validateTimeout(cfg.Telegram.APITimeout, ErrInvalidTelegramAPITimeout)
	if err != nil {
		return err
	}

	downloadTimeout, err := validateTimeout(cfg.Telegram.DownloadTimeout, ErrInvalidTelegramDownloadTimeout)
	if err != nil {
		return err
	}

	baseURL, err := url.Parse(strings.TrimSpace(cfg.STT.BaseURL))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidSTTBaseURL, err)
	}

	if baseURL.Scheme == "" || baseURL.Host == "" {
		return fmt.Errorf("%w: %q", ErrInvalidSTTBaseURL, cfg.STT.BaseURL)
	}

	sttTimeout, err := validateTimeout(cfg.STT.Timeout, ErrInvalidSTTTimeout)
	if err != nil {
		return err
	}

	cfg.telegramAPITimeout = apiTimeout
	cfg.telegramDownloadTimeout = downloadTimeout
	cfg.sttTimeout = sttTimeout

	return nil
}

// validateTimeout requires a parseable and positive duration: providers turn
// these values into context deadlines, and a non-positive deadline would
// expire before the request is even sent. The parsed value is returned so the
// caller can cache it instead of parsing the same string again.
func validateTimeout(raw string, sentinel error) (time.Duration, error) {
	parsed, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%w: %w", sentinel, err)
	}

	if parsed <= 0 {
		return 0, fmt.Errorf("%w: %q is not positive", sentinel, strings.TrimSpace(raw))
	}

	return parsed, nil
}
