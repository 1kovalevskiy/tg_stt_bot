package configs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const validConfigJSON = `{
  "app": { "log_level": "info" },
  "telegram": {
    "token": "12345:TEST_TOKEN",
    "admin_id": 100,
    "service_chat_id": -200,
    "allowed_chats": [-1001, 42]
  },
  "stt": { "base_url": "http://stt.local:5092/", "language": "ru", "timeout": "30s" }
}`

// configEnvVars is every variable the config reads.
var configEnvVars = []string{
	"APP_LOG_LEVEL",
	"TELEGRAM_TOKEN",
	"TELEGRAM_ADMIN_ID",
	"TELEGRAM_SERVICE_CHAT_ID",
	"TELEGRAM_ALLOWED_CHATS",
	"TELEGRAM_API_TIMEOUT",
	"TELEGRAM_DOWNLOAD_TIMEOUT",
	"STT_BASE_URL",
	"STT_LANGUAGE",
	"STT_TIMEOUT",
}

// clearConfigEnv makes the environment empty for the duration of the test.
// cleanenv reads the real process environment, so an ambient TELEGRAM_ADMIN_ID
// would not just break these tests: it would also mask a broken json tag by
// filling the field from somewhere else.
func clearConfigEnv(t *testing.T) {
	t.Helper()

	for _, name := range configEnvVars {
		// t.Setenv registers the restore; Unsetenv then makes the variable
		// actually absent, which an empty value would not be — cleanenv
		// applies anything that is merely set.
		t.Setenv(name, "")

		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}

func TestNewConfig_ValidFile(t *testing.T) {
	clearConfigEnv(t)

	path := writeConfigFile(t, validConfigJSON)

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig() unexpected error: %v", err)
	}

	if got := cfg.GetAppLogLevel(); got != "INFO" {
		t.Errorf("GetAppLogLevel() = %q, want %q", got, "INFO")
	}
	if got := cfg.GetTelegramToken(); got != "12345:TEST_TOKEN" {
		t.Errorf("GetTelegramToken() = %q, want %q", got, "12345:TEST_TOKEN")
	}
	if got := cfg.GetTelegramAdminID(); got != 100 {
		t.Errorf("GetTelegramAdminID() = %d, want %d", got, 100)
	}
	if got := cfg.GetTelegramServiceChatID(); got != -200 {
		t.Errorf("GetTelegramServiceChatID() = %d, want %d", got, -200)
	}
	if got := cfg.GetTelegramAllowedChats(); len(got) != 2 || got[0] != -1001 || got[1] != 42 {
		t.Errorf("GetTelegramAllowedChats() = %v, want %v", got, []int64{-1001, 42})
	}
	if got := cfg.GetSTTBaseURL(); got != "http://stt.local:5092" {
		t.Errorf("GetSTTBaseURL() = %q, want %q (trailing slash trimmed)", got, "http://stt.local:5092")
	}
	if got := cfg.GetSTTLanguage(); got != "ru" {
		t.Errorf("GetSTTLanguage() = %q, want %q", got, "ru")
	}
	if got := cfg.GetSTTTimeout(); got != 30*time.Second {
		t.Errorf("GetSTTTimeout() = %v, want %v", got, 30*time.Second)
	}

	// Not present in the file: the env defaults apply.
	if got := cfg.GetTelegramAPITimeout(); got != 30*time.Second {
		t.Errorf("GetTelegramAPITimeout() = %v, want default %v", got, 30*time.Second)
	}
	if got := cfg.GetTelegramDownloadTimeout(); got != 2*time.Minute {
		t.Errorf("GetTelegramDownloadTimeout() = %v, want default %v", got, 2*time.Minute)
	}
}

func TestNewConfig_EnvOverrides(t *testing.T) {
	clearConfigEnv(t)

	path := writeConfigFile(t, validConfigJSON)

	t.Setenv("APP_LOG_LEVEL", "debug")
	t.Setenv("TELEGRAM_TOKEN", "99999:ENV_TOKEN")
	t.Setenv("TELEGRAM_ADMIN_ID", "777")
	t.Setenv("TELEGRAM_SERVICE_CHAT_ID", "-888")
	t.Setenv("TELEGRAM_ALLOWED_CHATS", "-1,2,3")
	t.Setenv("TELEGRAM_API_TIMEOUT", "5s")
	t.Setenv("TELEGRAM_DOWNLOAD_TIMEOUT", "90s")
	t.Setenv("STT_BASE_URL", "http://192.168.10.53:5092")
	t.Setenv("STT_LANGUAGE", "en")
	t.Setenv("STT_TIMEOUT", "45s")

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig() unexpected error: %v", err)
	}

	if got := cfg.GetAppLogLevel(); got != "DEBUG" {
		t.Errorf("GetAppLogLevel() = %q, want %q", got, "DEBUG")
	}
	if got := cfg.GetTelegramToken(); got != "99999:ENV_TOKEN" {
		t.Errorf("GetTelegramToken() = %q, want %q", got, "99999:ENV_TOKEN")
	}
	if got := cfg.GetTelegramAdminID(); got != 777 {
		t.Errorf("GetTelegramAdminID() = %d, want %d", got, 777)
	}
	if got := cfg.GetTelegramServiceChatID(); got != -888 {
		t.Errorf("GetTelegramServiceChatID() = %d, want %d", got, -888)
	}
	if got := cfg.GetTelegramAllowedChats(); len(got) != 3 || got[0] != -1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("GetTelegramAllowedChats() = %v, want %v", got, []int64{-1, 2, 3})
	}
	if got := cfg.GetSTTBaseURL(); got != "http://192.168.10.53:5092" {
		t.Errorf("GetSTTBaseURL() = %q, want %q", got, "http://192.168.10.53:5092")
	}
	if got := cfg.GetSTTLanguage(); got != "en" {
		t.Errorf("GetSTTLanguage() = %q, want %q", got, "en")
	}
	if got := cfg.GetSTTTimeout(); got != 45*time.Second {
		t.Errorf("GetSTTTimeout() = %v, want %v", got, 45*time.Second)
	}
	if got := cfg.GetTelegramAPITimeout(); got != 5*time.Second {
		t.Errorf("GetTelegramAPITimeout() = %v, want %v", got, 5*time.Second)
	}
	if got := cfg.GetTelegramDownloadTimeout(); got != 90*time.Second {
		t.Errorf("GetTelegramDownloadTimeout() = %v, want %v", got, 90*time.Second)
	}
}

func TestNewConfig_MissingFile(t *testing.T) {
	clearConfigEnv(t)

	_, err := NewConfig(filepath.Join(t.TempDir(), "missing.json"))
	if !errors.Is(err, ErrReadConfig) {
		t.Fatalf("NewConfig() error = %v, want ErrReadConfig", err)
	}
}

func TestNewConfig_InvalidConfigReturnsValidationError(t *testing.T) {
	clearConfigEnv(t)

	path := writeConfigFile(t, `{
  "app": { "log_level": "INFO" },
  "telegram": { "token": "", "admin_id": 1, "service_chat_id": 2, "allowed_chats": [] },
  "stt": { "base_url": "http://stt.local", "language": "", "timeout": "10s" }
}`)

	_, err := NewConfig(path)
	if !errors.Is(err, ErrEmptyTelegramToken) {
		t.Fatalf("NewConfig() error = %v, want ErrEmptyTelegramToken", err)
	}
}

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(cfg *Config)
		wantErr error
	}{
		{
			name:    "valid config",
			mutate:  func(_ *Config) {},
			wantErr: nil,
		},
		{
			name:    "empty token",
			mutate:  func(cfg *Config) { cfg.Telegram.Token = "   " },
			wantErr: ErrEmptyTelegramToken,
		},
		{
			name:    "zero admin id",
			mutate:  func(cfg *Config) { cfg.Telegram.AdminID = 0 },
			wantErr: ErrZeroTelegramAdminID,
		},
		{
			name:    "zero service chat id",
			mutate:  func(cfg *Config) { cfg.Telegram.ServiceChatID = 0 },
			wantErr: ErrZeroTelegramServiceChatID,
		},
		{
			name:    "zero element in allowed chats",
			mutate:  func(cfg *Config) { cfg.Telegram.AllowedChats = []int64{-1001, 0} },
			wantErr: ErrZeroAllowedChat,
		},
		{
			name:    "unparseable base url",
			mutate:  func(cfg *Config) { cfg.STT.BaseURL = "http://bad host:port" },
			wantErr: ErrInvalidSTTBaseURL,
		},
		{
			name:    "base url without scheme",
			mutate:  func(cfg *Config) { cfg.STT.BaseURL = "stt.local:5092" },
			wantErr: ErrInvalidSTTBaseURL,
		},
		{
			name:    "empty base url",
			mutate:  func(cfg *Config) { cfg.STT.BaseURL = "" },
			wantErr: ErrInvalidSTTBaseURL,
		},
		{
			name:    "invalid timeout",
			mutate:  func(cfg *Config) { cfg.STT.Timeout = "not-a-duration" },
			wantErr: ErrInvalidSTTTimeout,
		},
		{
			name:    "non-positive stt timeout",
			mutate:  func(cfg *Config) { cfg.STT.Timeout = "0s" },
			wantErr: ErrInvalidSTTTimeout,
		},
		{
			name:    "invalid telegram api timeout",
			mutate:  func(cfg *Config) { cfg.Telegram.APITimeout = "not-a-duration" },
			wantErr: ErrInvalidTelegramAPITimeout,
		},
		{
			name:    "non-positive telegram api timeout",
			mutate:  func(cfg *Config) { cfg.Telegram.APITimeout = "-1s" },
			wantErr: ErrInvalidTelegramAPITimeout,
		},
		{
			name:    "invalid telegram download timeout",
			mutate:  func(cfg *Config) { cfg.Telegram.DownloadTimeout = "" },
			wantErr: ErrInvalidTelegramDownloadTimeout,
		},
		{
			name:    "non-positive telegram download timeout",
			mutate:  func(cfg *Config) { cfg.Telegram.DownloadTimeout = "0" },
			wantErr: ErrInvalidTelegramDownloadTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfigForValidation()
			tt.mutate(cfg)

			err := validateConfig(cfg)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("validateConfig() unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateConfig() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfig_NilConfig(t *testing.T) {
	t.Parallel()

	if err := validateConfig(nil); !errors.Is(err, ErrNilConfig) {
		t.Fatalf("validateConfig(nil) error = %v, want ErrNilConfig", err)
	}
}

func validConfigForValidation() *Config {
	return &Config{
		App: App{LogLevel: "INFO"},
		Telegram: Telegram{
			Token:           "12345:TEST_TOKEN",
			AdminID:         100,
			ServiceChatID:   -200,
			AllowedChats:    []int64{-1001, 42},
			APITimeout:      "30s",
			DownloadTimeout: "2m",
		},
		STT: STT{
			BaseURL: "http://stt.local:5092",
			Timeout: "120s",
		},
	}
}

// TestNewConfig_TimeoutsFromFile pins the json tags of the two timeout
// settings: they were only ever covered through their env defaults, so a
// typo'd tag would have gone unnoticed.
func TestNewConfig_TimeoutsFromFile(t *testing.T) {
	clearConfigEnv(t)

	path := writeConfigFile(t, `{
  "app": { "log_level": "info" },
  "telegram": {
    "token": "12345:TEST_TOKEN",
    "admin_id": 100,
    "service_chat_id": -200,
    "allowed_chats": [-1001],
    "api_timeout": "7s",
    "download_timeout": "3m"
  },
  "stt": { "base_url": "http://stt.local:5092", "language": "ru", "timeout": "11s" }
}`)

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig() unexpected error: %v", err)
	}

	if got := cfg.GetTelegramAPITimeout(); got != 7*time.Second {
		t.Errorf("GetTelegramAPITimeout() = %v, want %v", got, 7*time.Second)
	}

	if got := cfg.GetTelegramDownloadTimeout(); got != 3*time.Minute {
		t.Errorf("GetTelegramDownloadTimeout() = %v, want %v", got, 3*time.Minute)
	}

	if got := cfg.GetSTTTimeout(); got != 11*time.Second {
		t.Errorf("GetSTTTimeout() = %v, want %v", got, 11*time.Second)
	}
}

// TestNewConfig_IsNotAffectedByAmbientEnv guards the hermetic setup itself.
func TestNewConfig_IsNotAffectedByAmbientEnv(t *testing.T) {
	clearConfigEnv(t)

	path := writeConfigFile(t, validConfigJSON)

	t.Setenv("STT_LANGUAGE", "de")
	t.Setenv("TELEGRAM_ADMIN_ID", "999")

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig() unexpected error: %v", err)
	}

	// The environment wins over the file: that is the documented precedence.
	if got := cfg.GetSTTLanguage(); got != "de" {
		t.Errorf("GetSTTLanguage() = %q, want the env value %q", got, "de")
	}

	if got := cfg.GetTelegramAdminID(); got != 999 {
		t.Errorf("GetTelegramAdminID() = %d, want the env value %d", got, 999)
	}
}

func TestValidateConfig_CachesParsedTimeouts(t *testing.T) {
	t.Parallel()

	cfg := validConfigForValidation()

	// Before validation nothing is parsed: the getters must not silently
	// return a zero deadline, which would expire on the first call.
	if got := cfg.GetSTTTimeout(); got != 0 {
		t.Errorf("GetSTTTimeout() before validation = %v, want 0", got)
	}

	if err := validateConfig(cfg); err != nil {
		t.Fatalf("validateConfig() unexpected error: %v", err)
	}

	if got := cfg.GetSTTTimeout(); got != 120*time.Second {
		t.Errorf("GetSTTTimeout() = %v, want %v", got, 120*time.Second)
	}

	if got := cfg.GetTelegramAPITimeout(); got != 30*time.Second {
		t.Errorf("GetTelegramAPITimeout() = %v, want %v", got, 30*time.Second)
	}

	if got := cfg.GetTelegramDownloadTimeout(); got != 2*time.Minute {
		t.Errorf("GetTelegramDownloadTimeout() = %v, want %v", got, 2*time.Minute)
	}
}
