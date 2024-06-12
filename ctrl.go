package silly_ctrl

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"github.com/irealing/silly-ctrl/packet"
	"github.com/quic-go/quic-go"
	"google.golang.org/protobuf/encoding/protodelim"
	"net"
	"sort"
	"strings"
	"time"
)

type SessionExecCallback func(ctx context.Context, ret *packet.Ret, sess Session, stream quic.Stream) error
type Session interface {
	ID() string
	RemoteAddr() net.Addr
	IsRemote() bool // IsRemote 是否本地发起的连接
	App() *App
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
		err = UnknownCommandError
	}
	_, _ = protodelim.MarshalTo(stream, RetWithError(err))
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

type App struct {
	AccessKey string
	Secret    string
}

func (app *App) Signature() *packet.Handshake {
	return GenerateAuthToken(app.AccessKey, app.Secret)
}
func (app *App) Validate(handshake *packet.Handshake) error {
	delay := time.Now().Unix() - int64(handshake.T)
	if delay > 30 || delay < (-30) {
		return SignatureTimeoutError
	}
	if GenerateSignatureString(app.AccessKey, app.Secret, fmt.Sprintf("%d", handshake.T)) != handshake.Sign {
		return HandshakeFailedError
	}
	return nil
}
func GenerateSignature(args ...string) []byte {
	sort.Strings(args)
	val := strings.Join(args, "")
	h := sha256.New()
	h.Write([]byte(val))
	return h.Sum(nil)
}

func GenerateSignatureString(args ...string) string {
	return hex.EncodeToString(GenerateSignature(args...))
}

func GenerateAuthToken(ak, sk string) *packet.Handshake {
	t := time.Now().Unix()
	sign := GenerateSignatureString(ak, sk, fmt.Sprintf("%d", t))
	return &packet.Handshake{
		AccessKey: ak,
		Sign:      sign,
		T:         uint64(t),
	}
}

type Node interface {
	Run(ctx context.Context, tlsConfig *tls.Config) error
	Connect(ctx context.Context, addr string, app *App, tlsConfig *tls.Config) error
	Manager() SessionManager
}
type Validator interface {
	Validate(handshake *packet.Handshake) (*App, error)
}
