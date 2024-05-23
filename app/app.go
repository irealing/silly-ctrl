package main

import (
	"context"
	"flag"
	"github.com/irealing/silly-ctrl/app/config"
	"github.com/irealing/silly-ctrl/internal"
	"github.com/irealing/silly-ctrl/internal/ctrl/impl"
	"github.com/irealing/silly-ctrl/internal/util"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configFilename := flag.String("c", config.DefaultConfigFilename, "config file")
	flag.Parse()
	cfg, err := config.LoadConfig(*configFilename)
	if err != nil {
		slog.Error("load config file error", "err", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	startApp(ctx, cfg)
}
func startApp(ctx context.Context, cfg *config.Config) {
	node, err := impl.CreateNode(cfg.Logger(), &cfg.Ctrl, util.NewBasicValidator(cfg.Apps))
	if err != nil {
		cfg.Logger().Error("create node failed", "err", err)
		return
	}
	var wc []internal.WorkerCreator
	if len(cfg.Apps) > 0 {
		wc = append(wc, func(ctx context.Context) (internal.Worker, error) {
			return makeListenWorker(node, cfg)
		})
	}
	if len(cfg.Remote) > 0 {
		wc = append(wc, func(ctx context.Context) (internal.Worker, error) {
			return &remoteWorker{
				cfg:  cfg,
				node: node,
			}, nil
		})
	}
	if len(cfg.Forward) > 0 {
		wc = append(wc, func(ctx context.Context) (internal.Worker, error) {
			return &forwardWorker{
				cfg:  cfg,
				node: node,
			}, nil
		})
	}
	if err := internal.NewWorker(cfg.Logger(), "app", wc...).Run(ctx); err != nil {
		cfg.Logger().Error("app exit with error", "err", err)
	} else {
		cfg.Logger().Info("app exit")
	}
}
