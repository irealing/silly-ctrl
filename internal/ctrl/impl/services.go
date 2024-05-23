package impl

import (
	"context"
	"fmt"
	"github.com/irealing/silly-ctrl/internal/ctrl"
	"github.com/irealing/silly-ctrl/internal/util"
	"github.com/irealing/silly-ctrl/internal/util/packet"
	"github.com/quic-go/quic-go"
	"google.golang.org/protobuf/encoding/protodelim"
	"net"
)

type forwardService struct {
}

func (forward forwardService) Type() packet.CommandType {
	return packet.CommandType_FORWARD
}

func (forward forwardService) Invoke(ctx context.Context, command *packet.Command, _ ctrl.Session, manager ctrl.SessionManager, stream quic.Stream) error {
	if len(command.Args) < 2 {
		return util.BadParamError
	}
	remote, address := command.Args[0], command.Args[1]
	dest, ok := manager.Get(remote)
	if !ok {
		return util.UnknownSessionError
	}
	if !dest.IsRemote() {
		return forward.forwardClient(ctx, dest, stream, command)
	}
	return forward.forwardLocal(ctx, stream, command, address)
}
func (forward forwardService) forwardLocal(ctx context.Context, stream quic.Stream, cmd *packet.Command, address string) error {
	network := cmd.GetParamWithDefault("network", "tcp")
	conn, err := net.Dial(network, address)
	if err != nil {
		return fmt.Errorf("dial %s:%s", network, address)
	}
	defer func() {
		err = conn.Close()
	}()
	_, err = protodelim.MarshalTo(stream, util.RetWithError(util.NoError))
	if err != nil {
		return fmt.Errorf("write ret error %s", err)
	}
	return util.Forward(ctx, conn, stream)
}

func (forward forwardService) forwardClient(ctx context.Context, dest ctrl.Session, stream quic.Stream, cmd *packet.Command) error {
	return dest.Exec(cmd, func(_ *packet.Ret, sess ctrl.Session, remoteStream quic.Stream) error {
		if _, err := protodelim.MarshalTo(stream, util.RetWithError(util.NoError)); err != nil {
			return err
		}
		if err := util.Forward(ctx, remoteStream, stream); err != nil {
			return err
		}
		return nil
	})
}
