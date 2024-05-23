package impl

import (
	"context"
	"fmt"
	"github.com/irealing/silly-ctrl/internal/ctrl"
	"github.com/irealing/silly-ctrl/internal/util"
	"github.com/irealing/silly-ctrl/internal/util/packet"
	sillyKits "github.com/irealing/silly-kits"
	"github.com/quic-go/quic-go"
	"google.golang.org/protobuf/encoding/protodelim"
	"net"
	"time"
)

type forwardService struct {
}

func (forward forwardService) Type() packet.CommandType {
	return packet.CommandType_FORWARD
}

func (forward forwardService) Invoke(ctx context.Context, command *packet.Command, sess ctrl.Session, manager ctrl.SessionManager, stream quic.Stream) error {
	remote, isRemote := command.GetParam("remote")
	if isRemote {
		dest, ok := manager.Get(remote)
		if !ok {
			err := stream.SetWriteDeadline(time.Now().Add(time.Second * 20))
			if err != nil {
				return err
			}
			_, err = protodelim.MarshalTo(stream, util.RetWithError(err))
			return err
		}
		return forward.forwardRemote(ctx, dest, stream, command)
	}
	return forward.forwardLocal(ctx, stream, command)
}
func (forward forwardService) forwardLocal(ctx context.Context, stream quic.Stream, cmd *packet.Command) error {
	address, ok := cmd.GetParam("address")
	if !ok {
		return util.BadParamError
	}
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

func (forward forwardService) forwardRemote(ctx context.Context, dest ctrl.Session, stream quic.Stream, cmd *packet.Command) error {
	params := sillyKits.Filter(cmd.Params, func(param *packet.CommandParam) bool {
		return param.Key != "remote"
	})
	newCmd := &packet.Command{
		Type:   cmd.Type,
		Args:   cmd.Args,
		Params: params,
	}
	return dest.Exec(newCmd, func(_ *packet.Ret, sess ctrl.Session, remoteStream quic.Stream) error {
		return util.Forward(ctx, remoteStream, stream)
	})
}
