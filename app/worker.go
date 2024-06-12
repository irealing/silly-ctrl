package main

import (
	"context"
	"github.com/irealing/silly-ctrl"
	"github.com/irealing/silly-ctrl/app/config"
	"golang.org/x/sync/errgroup"
)

func makeListenWorker(node silly_ctrl.Node, cfg *config.Config) (silly_ctrl.Worker, error) {
	return silly_ctrl.SimpleWorker("listen", func(ctx context.Context) error {
		return node.Run(ctx, cfg.TLSConfig())
	}), nil
}

type remoteWorker struct {
	cfg  *config.Config
	node silly_ctrl.Node
}

func (worker *remoteWorker) Tag() string {
	return "remote"
}

func (worker *remoteWorker) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	for _, remote := range worker.cfg.Remote {
		eg.Go(func() error {
			return worker.runRemote(ctx, &remote)
		})
	}
	return eg.Wait()
}
func (worker *remoteWorker) runRemote(ctx context.Context, remote *config.Remote) error {
	for {
		select {
		case <-ctx.Done():
			worker.cfg.Logger().Info("remote done", "remote", remote.Address, "app", remote.App.AccessKey)
			return nil
		default:
		}
		if err := worker.node.Connect(ctx, remote.Address, &remote.App, worker.cfg.TLSConfig()); err != nil {
			worker.cfg.Logger().Error("remote connection error", "err", err, "remote", remote.Address, "app", remote.App.AccessKey)
		}
	}
}
