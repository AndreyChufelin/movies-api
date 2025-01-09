package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AndreyChufelin/movies-api/internal/config"
	"github.com/AndreyChufelin/movies-api/internal/logger"
	"github.com/AndreyChufelin/movies-api/internal/server/rest"
	"github.com/AndreyChufelin/movies-api/internal/storage/postgres"
)

func main() {
	defer exitHandler()
	logg := &logger.Logger{
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
	config, err := config.LoadConfig("configs/config-api.toml")
	if err != nil {
		logg.Fatal(
			"failed to load config",
			"error", err,
		)
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	shutCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()

	logg.Info("connecting to database")
	storage := postgres.NewStorage(
		config.DB.Host,
		config.DB.Port,
		config.DB.User,
		config.DB.Password,
		config.DB.Name,
	)
	err = storage.Connect(ctx)
	if err != nil {
		logg.Fatal(
			"falied to create connection with database",
			"error", err,
		)
	}
	defer storage.Close(ctx)

	restServer := rest.NewServer(
		logg,
		config.REST.Host,
		config.REST.Port,
		config.REST.IdleTimeout,
		config.REST.ReadTimeout,
		config.REST.WriteTimeout,
		storage,
		config.RateLimiter.Limit,
		config.RateLimiter.Enabled,
	)
	go func() {
		err = restServer.Start()
		if err != nil {
			logg.Fatal(
				"failed to start rest server",
				"error", err,
			)
		}
	}()
	defer func() {
		if err := restServer.Stop(shutCtx); err != nil {
			logg.Fatal("failed to stop rest server")
		}
	}()

	<-ctx.Done()
	logg.Info("stopping service")
}

func exitHandler() {
	if e := recover(); e != nil {
		if exit, ok := e.(logger.Exit); ok {
			os.Exit(exit.Code)
		}
		panic(e)
	}
}
