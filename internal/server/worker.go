package server

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/irealing/silly-ctrl/internal"
	sillyKits "github.com/irealing/silly-kits"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net/http"
)

type Worker interface {
	internal.Worker
	Mount(r gin.IRouter)
}
type WorkerCreator func(ctx context.Context, logger *slog.Logger) (Worker, error)

type BaseWorker struct {
	creators []WorkerCreator
	logger   *slog.Logger
	listen   string
}

func NewBaseWorker(logger *slog.Logger, listen string, creators ...WorkerCreator) internal.Worker {
	return &BaseWorker{creators: creators, logger: logger, listen: listen}
}

func (worker *BaseWorker) Tag() string {
	return "server"
}

func (worker *BaseWorker) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	workers, err := sillyKits.Map(worker.creators, func(t WorkerCreator) (Worker, error) {
		return t(ctx, worker.logger)
	})
	if err != nil {
		return err
	}
	engine := gin.Default()
	for _, w := range workers {
		router := engine.Group(fmt.Sprintf("/%s", w.Tag()))
		w.Mount(router)
	}
	eg.Go(func() error {
		return internal.RunWorkers(ctx, workers...)
	})
	worker.runHttpServer(ctx, eg, engine)
	return eg.Wait()
}
func (worker *BaseWorker) runHttpServer(ctx context.Context, eg *errgroup.Group, engine *gin.Engine) {
	srv := http.Server{Addr: worker.listen, Handler: engine}
	eg.Go(srv.ListenAndServe)
	eg.Go(func() error {
		<-ctx.Done()
		worker.logger.Warn("http context done")
		return srv.Shutdown(context.Background())
	})
}
