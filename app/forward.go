package main

import (
	"context"
	"github.com/irealing/silly-ctrl"
	"github.com/irealing/silly-ctrl/app/config"
	"github.com/irealing/silly-ctrl/packet"
	"github.com/quic-go/quic-go"
	"golang.org/x/sync/errgroup"
	"net"
	"sync"
)

type forwardWorker struct {
	cfg  *config.Config
	node silly_ctrl.Node
}

func (worker forwardWorker) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	for _, fc := range worker.cfg.Forward {
		cfg := fc
		eg.Go(func() error {
			return worker.forward(ctx, &cfg)
		})
	}
	return eg.Wait()
}
func (worker forwardWorker) listen(ctx context.Context, listener net.Listener) <-chan net.Conn {
	c := make(chan net.Conn)
	go func() {
		defer close(c)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case c <- conn:
			}
		}
	}()
	return c
}
func (worker forwardWorker) forward(ctx context.Context, remote *config.Forward) error {
	listener, err := net.Listen("tcp", remote.LocalAddress)
	if err != nil {
		return err
	}
	worker.cfg.Logger().Info("forward via local address", "address", remote.LocalAddress)
	cs := worker.listen(ctx, listener)
	wg := sync.WaitGroup{}
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		_ = listener.Close()
	}()
	for c := range cs {
		conn := c
		wg.Add(1)
		go worker.forwardConn(ctx, remote, conn, &wg)
	}
	return nil
}
func (worker forwardWorker) forwardConn(ctx context.Context, remote *config.Forward, conn net.Conn, wg *sync.WaitGroup) {
	defer func() {
		_ = conn.Close()
		wg.Done()
	}()
	via := remote.Via
	if via == "" {
		via = remote.App
	}
	sess, ok := worker.node.Manager().Get(via)
	if !ok {
		worker.cfg.Logger().Error("app offline", "app", remote.App)
		return
	}
	err := sess.Exec(ctx,
		packet.ForwardCommand(remote.App, remote.RemoteAddress),
		func(ctx context.Context, ret *packet.Ret, sess silly_ctrl.Session, stream quic.Stream) error {
			eg, ctx := errgroup.WithContext(ctx)
			eg.Go(func() error {
				defer stream.CancelRead(quic.StreamErrorCode(silly_ctrl.NoError))
				return silly_ctrl.CopyWithContext(ctx, conn, stream)
			})
			eg.Go(func() error {
				return silly_ctrl.CopyWithContext(ctx, stream, conn)
			})
			return eg.Wait()
		},
	)
	if err != nil {
		worker.cfg.Logger().Error("forward error", "remote", remote.App, "err", err)
	}
}
func (worker forwardWorker) Tag() string {
	return "forward"
}
