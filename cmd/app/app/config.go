package app

import (
	"fmt"

	"github.com/1kovalevskiy/tg_stt_bot/internal/configs"
)

// initConfig takes the config from the options or loads it from disk.
func (a *App) initConfig(opts *Opts) error {
	if opts != nil && opts.Config != nil {
		a.Config = opts.Config

		return nil
	}

	path := defaultConfigPath
	if opts != nil && opts.ConfigPath != "" {
		path = opts.ConfigPath
	}

	cfg, err := configs.NewConfig(path)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrReadConfig, err)
	}

	a.Config = cfg

	return nil
}
