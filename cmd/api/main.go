package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gab-mello/click-and-collect/internal/config"
	"github.com/gab-mello/click-and-collect/internal/orders"
	"github.com/gab-mello/click-and-collect/internal/server"
	"github.com/gab-mello/click-and-collect/internal/stores"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(cfg.LogLevel),
	}))
	slog.SetDefault(logger)

	storesRepo := stores.NewRepo()
	storesSvc := stores.NewService(storesRepo)
	storesH := stores.NewHandler(storesSvc)

	ordersRepo := orders.NewRepo()
	ordersSvc := orders.NewService(ordersRepo, storesSvc)
	ordersH := orders.NewHandler(ordersSvc)

	router := server.NewRouter(storesH, ordersH)
	srv := server.New(cfg, logger, router)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return srv.Run(ctx)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
