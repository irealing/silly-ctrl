package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)
	worker, err := createWorker(ctx)
	if err != nil {
		slog.Error("create worker failed", "error", err)
		cancel()
		return
	}
	if err := worker.Run(ctx); err != nil {
		slog.Error("run worker failed", "worker", worker.Tag(), "error", err)
	}
}
