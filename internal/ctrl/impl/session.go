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
	"log/slog"
	"net"
	"sync"
	"time"
)

type session struct {
	app           *util.App
	logger        *slog.Logger
	conn          quic.Connection
	heartbeat     packet.Heartbeat
	isRemote      bool
	handleMapping ctrl.ServiceMapping
	manager       ctrl.SessionManager
	cfg           *ctrl.Config
}

func (sess *session) IsRemote() bool {
	return sess.isRemote
}

func (sess *session) ID() string {
	return sess.app.AccessKey
}

func (sess *session) RemoteAddr() net.Addr {
	return sess.conn.RemoteAddr()
}

func (sess *session) App() *util.App {
	return sess.app
}

func (sess *session) Info() *packet.Heartbeat {
	return &sess.heartbeat
}

func (sess *session) run(ctx context.Context) error {
	defer func() {
		_ = sess.manager.Del(sess.ID())
	}()
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return sess.runHeartbeat(ctx)
	})
	eg.Go(func() error {
		return sess.start(ctx)
	})
	return eg.Wait()
}
func (sess *session) runHeartbeat(ctx context.Context) error {
	if sess.isRemote {
		return sess.sendHeartbeat(ctx)
	} else {
		return sess.receiveHeartbeat(ctx)
	}
}
func (sess *session) sendHeartbeat(ctx context.Context) error {
	sess.logger.Debug("start send heartbeat stream")
	stream, err := sess.conn.OpenUniStream()
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		if err = stream.Close(); err != nil {
			sess.logger.Warn("close send heartbeat stream error", "err", err)
		}
	}()
	maxHeartbeatInterval := time.Second * sess.cfg.MaxHeartbeatInterval
	ticker := time.NewTicker(time.Second * sess.cfg.HeartbeatInterval)
	defer ticker.Stop()
	for {
		sess.logger.Debug("write heartbeat message")
		if err := stream.SetWriteDeadline(time.Now().Add(maxHeartbeatInterval)); err != nil {
			sess.logger.Error("set write deadline error", "err", err, "session", sess.ID())
			return err
		}
		beat, err := packet.NewHeartbeat()
		if err != nil {
			sess.logger.Error("generate heartbeat message error", "err", err)
			return err
		}
		if _, err := protodelim.MarshalTo(stream, beat); err != nil {
			sess.logger.Error("write heartbeat error", "err", err)
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (sess *session) receiveHeartbeat(ctx context.Context) error {
	stream, err := sess.conn.AcceptUniStream(ctx)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		stream.CancelRead(quic.StreamErrorCode(util.UnknownError))
	}()
	ticker := time.NewTicker(sess.cfg.HeartbeatInterval)
	for {
		if err = stream.SetReadDeadline(time.Now().Add(time.Second * sess.cfg.MaxHeartbeatInterval)); err != nil {
			sess.logger.Error("set read deadline error", "err", err)
			return err
		}
		if err = protodelim.UnmarshalFrom(packet.NewProtoReader(stream), &sess.heartbeat); err != nil {
			sess.logger.Error("receive heartbeat error", "err", err)
			return err
		} else {
			sess.logger.Debug("receive heartbeat", "info", &sess.heartbeat)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (sess *session) start(ctx context.Context) error {
	wg := sync.WaitGroup{}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			stream, err := sess.conn.AcceptStream(ctx)
			if err != nil {
				return err
			}
			cmd := &packet.Command{}
			if err = protodelim.UnmarshalFrom(packet.NewProtoReader(stream), cmd); err != nil {
				return err
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := sess.handleCommand(ctx, cmd, stream); err != nil {
					sess.logger.Warn("handle command error", "cmd", cmd, "err", err)
				}
			}()
		}
	}
}
func (sess *session) handleCommand(ctx context.Context, cmd *packet.Command, stream quic.Stream) (err error) {
	defer func() {
		sess.logger.Debug("handle command over,close stream", "type", cmd.Type, "stream", stream.StreamID())
		if err := stream.Close(); err != nil {
			sess.logger.Error("close stream error", "err", err)
		}
	}()
	sess.logger.Debug("receive command", "type", cmd.Type, "session", sess.app.AccessKey)
	defer sess.logger.Debug("command invoke done", "type", cmd.Type, "session", sess.app.AccessKey, "err", err)
	err = sess.handleMapping.Invoke(ctx, cmd, sess, sess.manager, stream)
	return err
}
func (sess *session) Exec(ctx context.Context, cmd *packet.Command, callback ctrl.SessionExecCallback) error {
	stream, err := sess.conn.OpenStream()
	if err != nil {
		return fmt.Errorf("open stream error session %s err %w", sess.ID(), err)
	}
	defer func() {
		stream.CancelRead(quic.StreamErrorCode(util.NoError))
		sess.logger.Debug("close stream", "stream", stream.StreamID(), "cmd", cmd.Type)
		err = stream.Close()
		if err != nil {
			sess.logger.Warn("close stream error", "err", err, "sess", sess.ID())
		}
	}()
	_, err = protodelim.MarshalTo(stream, cmd)
	if err != nil {
		return fmt.Errorf("write command error %w", err)
	}
	var ret packet.Ret
	if err = protodelim.UnmarshalFrom(packet.NewProtoReader(stream), &ret); err != nil {
		return fmt.Errorf("read ret error %w", err)
	}
	if ret.ErrNo != util.NoError.Code() {
		return util.ErrorNo(ret.ErrNo)
	}
	if callback == nil {
		return nil
	}
	return callback(ctx, &ret, sess, stream)
}
