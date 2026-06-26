package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gab-mello/click-and-collect/internal/carts"
	"github.com/gab-mello/click-and-collect/internal/config"
	"github.com/gab-mello/click-and-collect/internal/db"
	"github.com/gab-mello/click-and-collect/internal/orders"
	"github.com/gab-mello/click-and-collect/internal/products"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	storesRepo := stores.NewRepo(pool)
	storesSvc := stores.NewService(storesRepo)
	storesH := stores.NewHandler(storesSvc)

	productsRepo := products.NewRepo(pool)
	productsSvc := products.NewService(productsRepo)
	productsH := products.NewHandler(productsSvc)

	ordersRepo := orders.NewRepo(pool)
	ordersSvc := orders.NewService(ordersRepo, storesSvc)
	ordersH := orders.NewHandler(ordersSvc)

	cartsRepo := carts.NewRepo(pool)
	cartsSvc := carts.NewService(cartsRepo, productsSvc, ordersSvc, pool)
	cartsH := carts.NewHandler(cartsSvc)

	router := server.NewRouter(storesH, productsH, cartsH, ordersH)
	srv := server.New(cfg, logger, router)

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
