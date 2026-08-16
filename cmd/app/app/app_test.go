package app

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/configs"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// testBotToken is a syntactically valid token; no request is ever sent with it.
const testBotToken = "123456:AAHtest-token"

const testConfigJSON = `{
  "app": { "log_level": "info" },
  "telegram": {
    "token": "123456:AAHtest-token",
    "admin_id": 555,
    "service_chat_id": -100777,
    "allowed_chats": [-100123]
  },
  "stt": { "base_url": "http://stt.local:5092", "language": "ru", "timeout": "30s" }
}`

// clearConfigEnv makes the environment empty for the duration of the test:
// the config reads the real process environment, and an ambient
// TELEGRAM_ADMIN_ID would silently replace what the test file says.
func clearConfigEnv(t *testing.T) {
	t.Helper()

	for _, name := range []string{
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
	} {
		t.Setenv(name, "")

		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
	}
}

// writeTestConfig writes a valid config file and returns its path.
func writeTestConfig(t *testing.T) string {
	t.Helper()

	clearConfigEnv(t)

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(testConfigJSON), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}

// testConfig loads the config the wiring tests run against.
func testConfig(t *testing.T) *configs.Config {
	t.Helper()

	cfg, err := configs.NewConfig(writeTestConfig(t))
	if err != nil {
		t.Fatalf("NewConfig() unexpected error: %v", err)
	}

	return cfg
}

// testBotClient builds a client without the getMe call, which needs network.
func testBotClient(t *testing.T) *bot.Bot {
	t.Helper()

	client, err := bot.New(testBotToken, bot.WithSkipGetMe())
	if err != nil {
		t.Fatalf("bot.New() unexpected error: %v", err)
	}

	return client
}

func TestInitConfig_UsesTheProvidedConfig(t *testing.T) {
	app := &App{}
	cfg := testConfig(t)

	if err := app.initConfig(&Opts{Config: cfg}); err != nil {
		t.Fatalf("initConfig() unexpected error: %v", err)
	}

	if app.Config != cfg {
		t.Error("initConfig() did not take the config from the options")
	}
}

func TestInitConfig_ReadsTheConfigPath(t *testing.T) {
	app := &App{}

	if err := app.initConfig(&Opts{ConfigPath: writeTestConfig(t)}); err != nil {
		t.Fatalf("initConfig() unexpected error: %v", err)
	}

	if app.Config == nil {
		t.Fatal("initConfig() left the config nil")
	}

	if got := app.Config.GetTelegramAdminID(); got != 555 {
		t.Errorf("GetTelegramAdminID() = %d, want 555", got)
	}
}

func TestInitConfig_MissingFile(t *testing.T) {
	app := &App{}

	err := app.initConfig(&Opts{ConfigPath: filepath.Join(t.TempDir(), "missing.json")})
	if !errors.Is(err, ErrReadConfig) {
		t.Fatalf("initConfig() error = %v, want ErrReadConfig", err)
	}
}

func TestInitConfig_NilOptsFallsBackToTheDefaultPath(t *testing.T) {
	app := &App{}

	// The default path does not exist in the test's working directory, which
	// is exactly what makes the fallback observable.
	if err := app.initConfig(nil); !errors.Is(err, ErrReadConfig) {
		t.Fatalf("initConfig(nil) error = %v, want ErrReadConfig", err)
	}
}

func TestInitSteps_RejectMissingDependencies(t *testing.T) {
	cfg := testConfig(t)

	tests := []struct {
		name    string
		app     *App
		step    func(*App) error
		wantErr error
	}{
		{
			name:    "logs without config",
			app:     &App{},
			step:    (*App).initLogs,
			wantErr: ErrNilConfig,
		},
		{
			name:    "bot client without config",
			app:     &App{},
			step:    (*App).initBotClient,
			wantErr: ErrNilConfig,
		},
		{
			name:    "bot client without log handler",
			app:     &App{Config: cfg},
			step:    (*App).initBotClient,
			wantErr: ErrNilLogHandler,
		},
		{
			name:    "providers without config",
			app:     &App{},
			step:    (*App).initProviders,
			wantErr: ErrNilConfig,
		},
		{
			name:    "providers without bot client",
			app:     &App{Config: cfg},
			step:    (*App).initProviders,
			wantErr: ErrNilBot,
		},
		{
			name:    "log sink without config",
			app:     &App{},
			step:    (*App).initLogSink,
			wantErr: ErrNilConfig,
		},
		{
			name:    "log sink without log handler",
			app:     &App{Config: cfg},
			step:    (*App).initLogSink,
			wantErr: ErrNilLogHandler,
		},
		{
			name:    "log sink without telegram provider",
			app:     &App{Config: cfg, logHandler: slog.Default().Handler()},
			step:    (*App).initLogSink,
			wantErr: ErrNilTelegramProvider,
		},
		{
			name:    "controllers without config",
			app:     &App{},
			step:    (*App).initControllers,
			wantErr: ErrNilConfig,
		},
		{
			name:    "controllers without telegram provider",
			app:     &App{Config: cfg},
			step:    (*App).initControllers,
			wantErr: ErrNilTelegramProvider,
		},
		{
			name:    "bot handlers without config",
			app:     &App{},
			step:    (*App).initBotHandlers,
			wantErr: ErrNilConfig,
		},
		{
			name:    "bot handlers without bot client",
			app:     &App{Config: cfg},
			step:    (*App).initBotHandlers,
			wantErr: ErrNilBot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.step(tt.app); !errors.Is(err, tt.wantErr) {
				t.Errorf("step error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestInitControllers_RequiresTheSTTProvider(t *testing.T) {
	app := &App{Config: testConfig(t), Bot: testBotClient(t)}

	if err := app.initProviders(); err != nil {
		t.Fatalf("initProviders() unexpected error: %v", err)
	}

	app.Providers.STT = nil

	if err := app.initControllers(); !errors.Is(err, ErrNilSTTProvider) {
		t.Errorf("initControllers() error = %v, want ErrNilSTTProvider", err)
	}
}

func TestInitBotHandlers_RequiresControllers(t *testing.T) {
	app := &App{Config: testConfig(t), Bot: testBotClient(t)}

	if err := app.initBotHandlers(); !errors.Is(err, ErrNilController) {
		t.Errorf("initBotHandlers() error = %v, want ErrNilController", err)
	}
}

func TestInitSteps_WireTheApplication(t *testing.T) {
	app := &App{Config: testConfig(t), Bot: testBotClient(t)}

	if err := app.initLogs(); err != nil {
		t.Fatalf("initLogs() unexpected error: %v", err)
	}

	if app.logHandler == nil {
		t.Fatal("initLogs() left the log handler nil")
	}

	if err := app.initProviders(); err != nil {
		t.Fatalf("initProviders() unexpected error: %v", err)
	}

	if app.Providers.STT == nil || app.Providers.Telegram == nil {
		t.Fatal("initProviders() left a provider nil")
	}

	if err := app.initLogSink(); err != nil {
		t.Fatalf("initLogSink() unexpected error: %v", err)
	}

	// The sink is the resource the app has to close on shutdown.
	if len(app.closers) != 1 {
		t.Fatalf("closers = %d, want the log sink registered", len(app.closers))
	}

	if err := app.initControllers(); err != nil {
		t.Fatalf("initControllers() unexpected error: %v", err)
	}

	if app.Controllers.Chat == nil || app.Controllers.Admin == nil {
		t.Fatal("initControllers() left a controller nil")
	}

	if err := app.initBotHandlers(); err != nil {
		t.Fatalf("initBotHandlers() unexpected error: %v", err)
	}

	if err := app.closeAll(); err != nil {
		t.Fatalf("closeAll() unexpected error: %v", err)
	}
}

func TestCloseAll_RunsClosersInReverseOrder(t *testing.T) {
	app := &App{}

	var order []string

	app.addCloser("first", func() error {
		order = append(order, "first")

		return nil
	})
	app.addCloser("second", func() error {
		order = append(order, "second")

		return nil
	})

	// A nil closer is dropped at registration, not at shutdown.
	app.addCloser("nil", nil)

	if len(app.closers) != 2 {
		t.Fatalf("closers = %d, want the nil one dropped", len(app.closers))
	}

	if err := app.closeAll(); err != nil {
		t.Fatalf("closeAll() unexpected error: %v", err)
	}

	if len(order) != 2 || order[0] != "second" || order[1] != "first" {
		t.Errorf("closers ran in order %v, want reverse registration order", order)
	}

	// Closing twice must not run the hooks again.
	if err := app.closeAll(); err != nil {
		t.Fatalf("second closeAll() unexpected error: %v", err)
	}

	if len(order) != 2 {
		t.Errorf("closers ran %d times in total, want 2", len(order))
	}
}

func TestCloseAll_JoinsFailures(t *testing.T) {
	app := &App{}

	firstErr := errors.New("sink drain timed out")
	secondErr := errors.New("db still busy")

	app.addCloser("sink", func() error { return firstErr })
	app.addCloser("db", func() error { return secondErr })

	closed := false

	app.addCloser("last", func() error {
		closed = true

		return nil
	})

	err := app.closeAll()
	if !errors.Is(err, ErrCloseResources) {
		t.Fatalf("closeAll() error = %v, want ErrCloseResources", err)
	}

	for _, want := range []error{firstErr, secondErr} {
		if !errors.Is(err, want) {
			t.Errorf("closeAll() error = %v, want it to wrap %v", err, want)
		}
	}

	// A failing closer must not stop the ones registered before it.
	if !closed {
		t.Error("a closer was skipped after an earlier failure")
	}
}

// registration is a single matcher/handler pair passed to the bot client.
type registration struct {
	match   bot.MatchFunc
	handler bot.HandlerFunc
}

// fakeRegistrar is a hand-written fake of the botRegistrar interface.
type fakeRegistrar struct {
	registrations []registration
}

func (f *fakeRegistrar) RegisterHandlerMatchFunc(
	matchFunc bot.MatchFunc, handler bot.HandlerFunc, _ ...bot.Middleware,
) string {
	f.registrations = append(f.registrations, registration{match: matchFunc, handler: handler})

	return "handler-id"
}

// TestRegisterHandlers_MatchersArePairedWithTheirHandlers pins the wiring
// itself: a matcher registered with the wrong handler would silently drop
// every update it matches, and no dispatcher-level test would notice.
func TestRegisterHandlers_MatchersArePairedWithTheirHandlers(t *testing.T) {
	tests := []struct {
		name   string
		update *tgmodels.Update
		want   func(*fakeChatController, *fakeAdminController) bool
	}{
		{
			name:   "voice",
			update: voiceUpdate(testAllowedID, 1),
			want: func(chat *fakeChatController, _ *fakeAdminController) bool {
				return len(chat.voice) == 1
			},
		},
		{
			name:   "video note",
			update: videoNoteUpdate(testAllowedID, 2),
			want: func(chat *fakeChatController, _ *fakeAdminController) bool {
				return len(chat.videoNote) == 1
			},
		},
		{
			name:   "admin command",
			update: textUpdate(testAdminID, "/status"),
			want: func(_ *fakeChatController, admin *fakeAdminController) bool {
				return len(admin.calls) == 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, chat, admin := newTestDispatcher()
			registrar := &fakeRegistrar{}

			registerHandlers(registrar, d)

			if len(registrar.registrations) != 3 {
				t.Fatalf("registered %d handlers, want 3", len(registrar.registrations))
			}

			matched := 0

			for _, reg := range registrar.registrations {
				if !reg.match(tt.update) {
					continue
				}

				matched++

				reg.handler(t.Context(), nil, tt.update)
			}

			if matched != 1 {
				t.Fatalf("%d matchers accepted the update, want exactly 1", matched)
			}

			if !tt.want(chat, admin) {
				t.Error("the matched handler did not call the controller the matcher stands for")
			}
		})
	}
}
