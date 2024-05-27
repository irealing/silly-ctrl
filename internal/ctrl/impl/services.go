package impl

import (
	"context"
	"fmt"
	"github.com/irealing/silly-ctrl/internal/ctrl"
	"github.com/irealing/silly-ctrl/internal/util"
	"github.com/irealing/silly-ctrl/internal/util/packet"
	"github.com/quic-go/quic-go"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protodelim"
	"net"
)

type forwardService struct {
}

func (forward forwardService) Type() packet.CommandType {
	return packet.CommandType_FORWARD
}

func (forward forwardService) Invoke(ctx context.Context, command *packet.Command, sess ctrl.Session, manager ctrl.SessionManager, stream quic.Stream) error {
	if len(command.Args) < 2 {
		return util.BadParamError
	}
	remote, address := command.Args[0], command.Args[1]

	dest, ok := manager.Get(remote)
	if !ok {
		return util.UnknownSessionError
	}
	newCmd := &packet.Command{
		Type:   packet.CommandType_PROXY,
		Args:   []string{address},
		Params: command.Params,
	}
	if sess.ID() == remote {
		return proxyService{}.Invoke(ctx, newCmd, sess, manager, stream)
	}
	return dest.Exec(ctx, newCmd, func(ctx context.Context, _ *packet.Ret, sess ctrl.Session, remoteStream quic.Stream) error {
		if _, err := protodelim.MarshalTo(stream, util.RetWithError(util.NoError)); err != nil {
			return err
		}
		if err := util.Forward(ctx, remoteStream, stream); err != nil {
			return err
		}
		return nil
	})
}

type proxyService struct {
}

func (proxy proxyService) Type() packet.CommandType {
	return packet.CommandType_PROXY
}

func (proxy proxyService) Invoke(ctx context.Context, command *packet.Command, _ ctrl.Session, _ ctrl.SessionManager, stream quic.Stream) error {
	if len(command.Args) < 1 {
		return util.BadParamError
	}
	address := command.Args[0]
	network := command.GetParamWithDefault("network", "tcp")
	conn, err := net.Dial(network, address)
	if err != nil {
		return fmt.Errorf("dial %s:%s", network, address)
	}
	_, err = protodelim.MarshalTo(stream, util.RetWithError(util.NoError))
	if err != nil {
		return fmt.Errorf("write ret error %s", err)
	}
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		<-ctx.Done()
		_ = conn.Close()
		stream.CancelRead(quic.StreamErrorCode(util.NoError))
		return stream.Close()
	})
	eg.Go(func() error {
		return util.CopyWithContext(ctx, stream, conn)
	})
	eg.Go(func() error {
		return util.CopyWithContext(ctx, conn, stream)
	})
	return eg.Wait()
}
