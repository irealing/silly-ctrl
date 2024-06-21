package main

import (
	"context"
	"flag"
	"github.com/irealing/silly-ctrl"
	"github.com/irealing/silly-ctrl/app/config"
	"github.com/irealing/silly-ctrl/impl"
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
	node, err := impl.CreateNode(cfg.Logger(), &cfg.Ctrl, silly_ctrl.NewBasicValidator(cfg.Apps), impl.DefaultServices())
	if err != nil {
		cfg.Logger().Error("create node failed", "err", err)
		return
	}
	var wc []silly_ctrl.WorkerCreator
	if len(cfg.Apps) > 0 {
		wc = append(wc, func(ctx context.Context) (silly_ctrl.Worker, error) {
			return makeListenWorker(node, cfg)
		})
	}
	if len(cfg.Remote) > 0 {
		wc = append(wc, func(ctx context.Context) (silly_ctrl.Worker, error) {
			return &remoteWorker{
				cfg:  cfg,
				node: node,
			}, nil
		})
	}
	if len(cfg.Forward) > 0 {
		wc = append(wc, func(ctx context.Context) (silly_ctrl.Worker, error) {
			return &forwardWorker{
				cfg:  cfg,
				node: node,
			}, nil
		})
	}
	if err := silly_ctrl.NewWorker(cfg.Logger(), "app", wc...).Run(ctx); err != nil {
		cfg.Logger().Error("app exit with error", "err", err)
	} else {
		cfg.Logger().Info("app exit")
	}
}
