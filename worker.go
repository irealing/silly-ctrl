package silly_ctrl

import (
	"context"
	sillyKits "github.com/irealing/silly-kits"
	"golang.org/x/sync/errgroup"
	"log/slog"
)

type baseWorker struct {
	logger   *slog.Logger
	name     string
	creators []WorkerCreator
}

func NewWorker(logger *slog.Logger, name string, creators ...WorkerCreator) Worker {
	return &baseWorker{logger: logger, name: name, creators: creators}
}
func (worker *baseWorker) Tag() string {
	return worker.name
}

func (worker *baseWorker) Run(ctx context.Context) error {
	workers, err := sillyKits.Map(worker.creators, func(creator WorkerCreator) (Worker, error) {
		return creator(ctx)
	})
	if err != nil {
		slog.Warn("create workers failed", "error", err)
		return err
	}
	return RunWorkers(ctx, workers...)
}

type simpleWorker struct {
	tag string
	fn  func(ctx context.Context) error
}

func SimpleWorker(tag string, fn func(ctx context.Context) error) Worker {
	return &simpleWorker{tag: tag, fn: fn}
}

func (worker *simpleWorker) Run(ctx context.Context) error {
	if worker.fn != nil {
		return worker.fn(ctx)
	}
	return nil
}
func (worker *simpleWorker) Tag() string {
	return worker.tag
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
