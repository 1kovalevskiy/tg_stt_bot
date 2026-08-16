// Package app wires the application together: config, logging, providers,
// controllers and the Telegram long polling client. It is the only place that
// knows the concrete types behind the interfaces the layers consume.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/1kovalevskiy/tg_stt_bot/internal/configs"
	adminController "github.com/1kovalevskiy/tg_stt_bot/internal/controllers/admin-controller"
	chatController "github.com/1kovalevskiy/tg_stt_bot/internal/controllers/chat-controller"
	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	sttProvider "github.com/1kovalevskiy/tg_stt_bot/internal/providers/stt"
	telegramProvider "github.com/1kovalevskiy/tg_stt_bot/internal/providers/telegram"
	"github.com/go-telegram/bot"
)

type (
	// appCloser is a named shutdown hook run in reverse registration order.
	appCloser struct {
		name  string
		close func() error
	}

	// Providers holds the concrete provider implementations.
	Providers struct {
		STT      *sttProvider.Provider
		Telegram *telegramProvider.Provider
	}

	// Controllers holds the concrete controller implementations.
	Controllers struct {
		Chat  *chatController.Controller
		Admin *adminController.Controller
	}

	// App is the composition root of the bot.
	App struct {
		Config      *configs.Config
		Providers   Providers
		Controllers Controllers
		Bot         *bot.Bot

		logHandler slog.Handler
		closers    []appCloser
	}

	// Opts allows the caller to override what InitApp would build itself.
	Opts struct {
		ConfigPath string
		Config     *configs.Config
	}
)

// InitApp builds the application step by step and panics if any step fails:
// a half-wired bot is not worth running.
//
// The bot client is created before the providers because the Telegram
// provider is built on top of it, and its update handlers are registered
// last, once the controllers they dispatch to exist.
func InitApp(opts *Opts) *App {
	app := &App{}

	steps := []struct {
		name string
		init func() error
	}{
		{name: "config", init: func() error { return app.initConfig(opts) }},
		{name: "logs", init: app.initLogs},
		{name: "bot client", init: app.initBotClient},
		{name: "providers", init: app.initProviders},
		{name: "log sink", init: app.initLogSink},
		{name: "controllers", init: app.initControllers},
		{name: "bot handlers", init: app.initBotHandlers},
	}

	for _, step := range steps {
		if err := step.init(); err != nil {
			panic(fmt.Errorf("init %s: %w", step.name, err))
		}
	}

	return app
}

// RunApp starts long polling and blocks until an interrupt signal arrives,
// then closes every registered resource.
func (a *App) RunApp(ctx context.Context) {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	defer func() {
		if err := a.closeAll(); err != nil {
			slog.Error("failed to close app resources", "err", err)
		}
	}()

	slog.Info("bot started", "workers", models.BotWorkers)

	// Start returns when ctx is canceled, i.e. on a shutdown signal.
	a.Bot.Start(ctx)

	slog.Info("shutdown signal received, closing app resources")
}

// addCloser registers a shutdown hook.
func (a *App) addCloser(name string, closeFn func() error) {
	if closeFn == nil {
		return
	}

	a.closers = append(a.closers, appCloser{name: name, close: closeFn})
}

// closeAll runs the registered hooks in reverse order. It is safe to call
// more than once: the hooks are dropped as they run.
func (a *App) closeAll() error {
	var closeErr error

	closers := a.closers
	a.closers = nil

	for i := len(closers) - 1; i >= 0; i-- {
		closer := closers[i]

		// A nil hook is dropped by addCloser, so every closer here is callable.
		if err := closer.close(); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("%s: %w", closer.name, err))
		}
	}

	if closeErr != nil {
		return fmt.Errorf("%w: %w", ErrCloseResources, closeErr)
	}

	return nil
}
