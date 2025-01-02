package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/AndreyChufelin/movies-api/internal/config"
	"github.com/AndreyChufelin/movies-api/internal/server/rest"
	"github.com/AndreyChufelin/movies-api/internal/storage/postgres"
)

func main() {
	config, err := config.LoadConfig("configs/config-api.toml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to load config: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	storage := postgres.NewStorage(
		config.DB.Host,
		config.DB.Port,
		config.DB.User,
		config.DB.Password,
		config.DB.Name,
	)
	err = storage.Connect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection with database: %v\n", err)
		os.Exit(1)
	}
	defer storage.Close(ctx)

	restServer := rest.NewServer(
		config.REST.Host,
		config.REST.Port,
		config.REST.IdleTimeout,
		config.REST.ReadTimeout,
		config.REST.WriteTimeout,
		storage,
	)
	err = restServer.Start()
	if err != nil {
		slog.Error("failed to start rest server", "error", err)
	}
}
