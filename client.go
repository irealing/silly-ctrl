//go:build !server

package main

import (
	"context"
	"github.com/irealing/silly-ctrl/internal"
	"log/slog"
)

func createWorker(_ context.Context) (internal.Worker, error) {
	return internal.NewWorker(slog.Default(), "main"), nil
}
