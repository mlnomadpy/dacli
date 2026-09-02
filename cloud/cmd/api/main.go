package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/mlnomadpy/dacli/cloud/internal/config"
	"github.com/mlnomadpy/dacli/cloud/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "control-plane API stopped:", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "cloud/config.development.json", "path to strict JSON configuration")
	flag.Parse()
	cfg, err := config.LoadFile(*configPath, nil)
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	logger.Info("starting control-plane API", "config", cfg.SafeSummary())
	databaseAddress, err := cfg.DatabaseAddress()
	if err != nil {
		return err
	}
	readiness := service.ReadinessFunc(func(ctx context.Context) error {
		connection, dialErr := (&net.Dialer{Timeout: cfg.RequestTimeout}).DialContext(ctx, "tcp", databaseAddress)
		if dialErr != nil {
			return dialErr
		}
		return connection.Close()
	})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return service.RunAPI(ctx, cfg, service.NewAPI(cfg, readiness, logger))
}
