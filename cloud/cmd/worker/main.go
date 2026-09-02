package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mlnomadpy/dacli/cloud/internal/config"
	"github.com/mlnomadpy/dacli/cloud/internal/domain"
	"github.com/mlnomadpy/dacli/cloud/internal/worker"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "control-plane worker stopped:", err)
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
	logger.Info("starting control-plane worker", "component", domain.ComponentWorker, "contract_version", domain.ContractVersion, "config", cfg.SafeSummary())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	worker.Run(ctx, cfg.WorkerInterval, cfg.RequestTimeout, worker.JobFunc(func(context.Context) error {
		// Durable queue consumption is intentionally introduced by #988. This
		// lifecycle process is runnable now without claiming that work exists.
		return nil
	}), func(err error) {
		logger.Error("control-plane worker cycle failed", "code", "worker_cycle_failed")
	})
	return nil
}
