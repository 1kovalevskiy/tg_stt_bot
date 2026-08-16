package main

import (
	"context"
	"flag"

	"github.com/1kovalevskiy/tg_stt_bot/cmd/app/app"
	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

func main() {
	configPath := flag.String("config", models.DefaultConfigPath, "path to the config file")
	flag.Parse()

	ctx := context.Background()

	application := app.InitApp(ctx, &app.Opts{ConfigPath: *configPath})

	application.RunApp(ctx)
}
