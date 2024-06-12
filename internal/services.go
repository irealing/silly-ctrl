package internal

import (
	"context"
	"fmt"
	"github.com/irealing/silly-ctrl"
	"github.com/irealing/silly-ctrl/packet"
	"github.com/quic-go/quic-go"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protodelim"
	"net"
	"os/exec"
	"strings"
)

type forwardService struct {
}

func (forward forwardService) Type() packet.CommandType {
	return packet.CommandType_FORWARD
}

func (forward forwardService) Invoke(ctx context.Context, command *packet.Command, sess silly_ctrl.Session, manager silly_ctrl.SessionManager, stream quic.Stream) error {
	if len(command.Args) < 2 {
		return silly_ctrl.BadParamError
	}
	remote, address := command.Args[0], command.Args[1]

	dest, ok := manager.Get(remote)
	if !ok {
		return silly_ctrl.UnknownSessionError
	}
	newCmd := &packet.Command{
		Type:   packet.CommandType_PROXY,
		Args:   []string{address},
		Params: command.Params,
	}
	if sess.ID() == remote {
		return proxyService{}.Invoke(ctx, newCmd, sess, manager, stream)
	}
	return dest.Exec(ctx, newCmd, func(ctx context.Context, _ *packet.Ret, sess silly_ctrl.Session, remoteStream quic.Stream) error {
		if _, err := protodelim.MarshalTo(stream, silly_ctrl.RetWithError(silly_ctrl.NoError)); err != nil {
			return err
		}
		if err := silly_ctrl.Forward(ctx, remoteStream, stream); err != nil {
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

func (proxy proxyService) Invoke(ctx context.Context, command *packet.Command, _ silly_ctrl.Session, _ silly_ctrl.SessionManager, stream quic.Stream) error {
	if len(command.Args) < 1 {
		return silly_ctrl.BadParamError
	}
	address := command.Args[0]
	network := command.GetParamWithDefault("network", "tcp")
	conn, err := net.Dial(network, address)
	if err != nil {
		return fmt.Errorf("dial %s:%s", network, address)
	}
	_, err = protodelim.MarshalTo(stream, silly_ctrl.RetWithError(silly_ctrl.NoError))
	if err != nil {
		return fmt.Errorf("write ret error %s", err)
	}
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		<-ctx.Done()
		_ = conn.Close()
		stream.CancelRead(quic.StreamErrorCode(silly_ctrl.NoError))
		return stream.Close()
	})
	eg.Go(func() error {
		return silly_ctrl.CopyWithContext(ctx, stream, conn)
	})
	eg.Go(func() error {
		return silly_ctrl.CopyWithContext(ctx, conn, stream)
	})
	return eg.Wait()
}

type execService struct {
}

func (execService) Type() packet.CommandType {
	return packet.CommandType_EXEC
}

func (execService) Invoke(ctx context.Context, command *packet.Command, _ silly_ctrl.Session, _ silly_ctrl.SessionManager, stream quic.Stream) error {
	if len(command.Args) < 1 {
		return silly_ctrl.BadParamError
	}
	if _, err := protodelim.MarshalTo(stream, silly_ctrl.RetWithError(silly_ctrl.NoError)); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, command.Args[0], command.Args[1:]...)
	cmd.Path = command.GetParamWithDefault("dir", ".")
	cmd.Env = strings.Split(command.GetParamWithDefault("env", ""), ";")
	cmd.Stdout = stream
	cmd.Stderr = stream
	return cmd.Run()
}

type emptyService struct {
}

func (emptyService) Type() packet.CommandType {
	return packet.CommandType_EMPTY
}

func (emptyService) Invoke(_ context.Context, _ *packet.Command, _ silly_ctrl.Session, _ silly_ctrl.SessionManager, _ quic.Stream) error {
	return silly_ctrl.NoError
}
