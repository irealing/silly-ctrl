//go:build server

package main

import (
	"context"
	"github.com/irealing/silly-ctrl/internal"
	"github.com/irealing/silly-ctrl/internal/server"
	"log/slog"
)

func createWorker(_ context.Context) (internal.Worker, error) {
	return server.NewBaseWorker(slog.Default(), "127.0.0.1:8000"), nil
}
