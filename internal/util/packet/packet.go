//go:generate protoc -I=./ --go_out=. ./packet.proto
package packet

import (
	sillyKits "github.com/irealing/silly-kits"
	"github.com/quic-go/quic-go"
	"google.golang.org/protobuf/encoding/protodelim"
	"os"
	"os/user"
	"runtime"
	"time"
)

type protoReader struct {
	stream quic.ReceiveStream
}

func NewProtoReader(stream quic.ReceiveStream) protodelim.Reader {
	return &protoReader{stream: stream}
}

func (reader *protoReader) Read(p []byte) (n int, err error) {
	return reader.stream.Read(p)
}

func (reader *protoReader) ReadByte() (byte, error) {
	var b [1]byte
	_, err := reader.stream.Read(b[:])
	return b[0], err
}

func NewHeartbeat() (*Heartbeat, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	username, err := user.Current()
	if err != nil {
		return nil, err
	}
	return &Heartbeat{
		Hostname:  hostname,
		Username:  username.Username,
		OsName:    runtime.GOOS,
		Localtime: time.Now().Unix(),
	}, nil
}
func (x *Command) GetParam(key string) (string, bool) {
	_, param, err := sillyKits.Find(x.Params, func(param *CommandParam) bool {
		return param.Key == key
	})
	if err != nil {
		return "", false
	}
	return param.Value, true
}
func (x *Command) GetParamWithDefault(key, val string) string {
	r, ok := x.GetParam(key)
	if !ok {
		r = val
	}
	return r
}

// ForwardCommand FORWARD <REMOTE> <ADDRESS>
// like FORWARD xxx 127.0.0.1:8000
func ForwardCommand(remote, addr string) *Command {
	return &Command{
		Type: CommandType_FORWARD,
		Args: []string{remote, addr},
	}
}
