package main

import (
	"context"
	"flag"

	"github.com/1kovalevskiy/tg_stt_bot/cmd/app/app"
)

func main() {
	configPath := flag.String("config", "config.json", "path to the config file")
	flag.Parse()

	ctx := context.Background()

	application := app.InitApp(ctx, &app.Opts{ConfigPath: *configPath})

	application.RunApp(ctx)
}
