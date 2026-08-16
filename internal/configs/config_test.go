package config

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

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}

func TestNewConfig_ValidFile(t *testing.T) {
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
}

func TestNewConfig_EnvOverrides(t *testing.T) {
	path := writeConfigFile(t, validConfigJSON)

	t.Setenv("APP_LOG_LEVEL", "debug")
	t.Setenv("TELEGRAM_TOKEN", "99999:ENV_TOKEN")
	t.Setenv("TELEGRAM_ADMIN_ID", "777")
	t.Setenv("TELEGRAM_SERVICE_CHAT_ID", "-888")
	t.Setenv("TELEGRAM_ALLOWED_CHATS", "-1,2,3")
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
}

func TestNewConfig_MissingFile(t *testing.T) {
	_, err := NewConfig(filepath.Join(t.TempDir(), "missing.json"))
	if !errors.Is(err, ErrReadConfig) {
		t.Fatalf("NewConfig() error = %v, want ErrReadConfig", err)
	}
}

func TestNewConfig_InvalidConfigReturnsValidationError(t *testing.T) {
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
			Token:         "12345:TEST_TOKEN",
			AdminID:       100,
			ServiceChatID: -200,
			AllowedChats:  []int64{-1001, 42},
		},
		STT: STT{
			BaseURL: "http://stt.local:5092",
			Timeout: "120s",
		},
	}
}
