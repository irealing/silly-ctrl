package silly_ctrl

import (
	"context"
	"crypto/tls"
	"github.com/irealing/silly-ctrl/internal/util"
	"github.com/irealing/silly-ctrl/internal/util/packet"
	"github.com/quic-go/quic-go"
	"google.golang.org/protobuf/encoding/protodelim"
	"net"
)

type SessionExecCallback func(ctx context.Context, ret *packet.Ret, sess Session, stream quic.Stream) error
type Session interface {
	ID() string
	RemoteAddr() net.Addr
	IsRemote() bool // IsRemote 是否本地发起的连接
	App() *util.App
	Info() *packet.Heartbeat
	Exec(ctx context.Context, cmd *packet.Command, callback SessionExecCallback) error
}

type SessionManager interface {
	Put(sess Session) error
	Get(accessKey string) (Session, bool)
	Del(accessKey string) error
}

type Service interface {
	Type() packet.CommandType
	Invoke(ctx context.Context, command *packet.Command, session Session, manager SessionManager, stream quic.Stream) error
}
type ServiceMapping map[packet.CommandType]Service

func (mapping ServiceMapping) Invoke(ctx context.Context, cmd *packet.Command, session Session, manager SessionManager, stream quic.Stream) error {
	var err error
	if service, ok := mapping[cmd.Type]; ok {
		err = service.Invoke(ctx, cmd, session, manager, stream)
	} else {
		err = util.UnknownCommandError
	}
	_, _ = protodelim.MarshalTo(stream, util.RetWithError(err))
	return err
}

func (mapping ServiceMapping) Type() packet.CommandType {
	return packet.CommandType_EMPTY
}

func (mapping ServiceMapping) Register(services ...Service) ServiceMapping {
	for _, service := range services {
		mapping[service.Type()] = service
	}
	return mapping
}

type Node interface {
	Run(ctx context.Context, tlsConfig *tls.Config) error
	Connect(ctx context.Context, addr string, app *util.App, tlsConfig *tls.Config) error
	Manager() SessionManager
}
