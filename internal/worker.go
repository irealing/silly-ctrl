package internal

import (
	"context"
	sillyKits "github.com/irealing/silly-kits"
	"golang.org/x/sync/errgroup"
	"log/slog"
)

type simpleWorker struct {
	logger   *slog.Logger
	name     string
	creators []WorkerCreator
}

func NewWorker(logger *slog.Logger, name string, creators ...WorkerCreator) Worker {
	return &simpleWorker{logger: logger, name: name, creators: creators}
}
func (worker *simpleWorker) Tag() string {
	return worker.name
}

func (worker *simpleWorker) Run(ctx context.Context) error {
	workers, err := sillyKits.Map(worker.creators, func(creator WorkerCreator) (Worker, error) {
		return creator(ctx)
	})
	if err != nil {
		slog.Warn("create workers failed", "error", err)
		return err
	}
	return RunWorkers(ctx, workers...)
}

func RunWorkers[T Worker](ctx context.Context, workers ...T) error {
	eg, ctx := errgroup.WithContext(ctx)
	for _, worker := range workers {
		w := worker
		eg.Go(func() error {
			return w.Run(ctx)
		})
	}
	return eg.Wait()
}
