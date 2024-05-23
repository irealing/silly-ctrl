package ctrl

import (
	"context"
	"crypto/tls"
	"github.com/irealing/silly-ctrl/internal/util"
	"github.com/irealing/silly-ctrl/internal/util/packet"
	"github.com/quic-go/quic-go"
	"google.golang.org/protobuf/encoding/protodelim"
	"net"
)

type SessionExecCallback func(ret *packet.Ret, sess Session, stream quic.Stream) error
type Session interface {
	ID() string
	RemoteAddr() net.Addr
	App() *util.App
	Info() *packet.Heartbeat
	Exec(cmd *packet.Command, callback SessionExecCallback) error
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

func (mapping ServiceMapping) Invoke(ctx context.Context, _ *packet.Command, session Session, manager SessionManager, stream quic.Stream) error {
	cmd := &packet.Command{}
	err := protodelim.UnmarshalFrom(packet.NewProtoReader(stream), cmd)
	if err != nil {
		return err
	}
	if service, ok := mapping[cmd.Type]; ok {
		return service.Invoke(ctx, cmd, session, manager, stream)
	} else {
		return util.UnknownCommandError
	}
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
