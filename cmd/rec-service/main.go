package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/osamikoyo/recomendation-service/app"
	"github.com/osamikoyo/recomendation-service/config"
	"github.com/osamikoyo/recomendation-service/logger"
	"go.uber.org/zap"
)

func main() {
	logger.Init(logger.Config{
		LogFile:   "logs/rec-service.log",
		LogLevel:  "debug",
		AppName:   "rec-service",
		AddCaller: false,
	})

	logger := logger.Get()

	logger.Info("starting rec-service")

	path := ""
	for i, arg := range os.Args {
		if arg == "--config" {
			path = os.Args[i+1]
		}
	}

	cfg, err := config.LoadConfig(path)
	if err != nil {
		logger.Error("failed load config",
			zap.String("path", path),
			zap.Error(err))
		return
	}

	app, err := app.SetupApp(logger, cfg)
	if err != nil {
		logger.Error("failed setup app",
			zap.Error(err))

		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	go func() {
		<-ctx.Done()

		logger.Info("stopping app")

		app.Close(ctx)
	}()

	if err = app.Run(ctx); err != nil {
		logger.Error("failed start app",
			zap.Error(err))

		return
	}
}
