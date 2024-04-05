package client

import (
	"context"
	"github.com/irealing/silly-ctrl/internal"
	"log/slog"
)

type clientWorker struct {
	logger *slog.Logger
}

func NewClientWorker(_ context.Context) (internal.Worker, error) {
	return &clientWorker{logger: slog.Default()}, nil
}

func (w *clientWorker) Tag() string {
	return "client"
}

func (w *clientWorker) Run(ctx context.Context) error {
	endpoint := &Endpoint{logger: w.logger}
	return endpoint.Run(ctx)
}
