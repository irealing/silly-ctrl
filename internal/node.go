package internal

import (
	"context"
	"crypto/tls"
	"github.com/irealing/silly-ctrl"
	"github.com/irealing/silly-ctrl/packet"
	"github.com/quic-go/quic-go"
	"google.golang.org/protobuf/encoding/protodelim"
	"log/slog"
	"net"
	"sync"
	"time"
)

type ctrlNode struct {
	logger         *slog.Logger
	tr             *quic.Transport
	manager        silly_ctrl.SessionManager
	valid          silly_ctrl.Validator
	serviceMapping silly_ctrl.ServiceMapping
	quicConfig     quic.Config
	cfg            *silly_ctrl.Config
}

func CreateNode(logger *slog.Logger, cfg *silly_ctrl.Config, valid silly_ctrl.Validator, services silly_ctrl.ServiceMapping) (silly_ctrl.Node, error) {
	if cfg == nil {
		cfg = silly_ctrl.DefaultConfig()
	}
	udpAddr, err := net.ResolveUDPAddr("udp", cfg.LocalAddress)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	return &ctrlNode{
		logger:  logger,
		tr:      &quic.Transport{Conn: conn},
		manager: NewManager(),
		valid:   valid,
		cfg:     cfg,
		quicConfig: quic.Config{
			KeepAlivePeriod: time.Second * cfg.MaxHeartbeatInterval,
			MaxIdleTimeout:  time.Second * cfg.MaxHeartbeatInterval * 2,
		},
		serviceMapping: services,
	}, nil
}

func (server *ctrlNode) Run(ctx context.Context, tlsConfig *tls.Config) error {
	listener, err := server.tr.Listen(tlsConfig, &server.quicConfig)
	if err != nil {
		return err
	}
	connections := server.accept(ctx, listener)
	server.start(ctx, connections)
	return err
}
func (server *ctrlNode) accept(ctx context.Context, listener *quic.Listener) <-chan quic.Connection {
	connections := make(chan quic.Connection, server.cfg.ConnectionQueueSize)
	go func() {
		defer close(connections)
		for {
			conn, err := listener.Accept(ctx)
			if err != nil {
				server.logger.Error("accept connection error", "err", err, "module", "ctrlNode")
				return
			}
			connections <- conn
		}
	}()
	return connections
}
func (server *ctrlNode) start(ctx context.Context, connections <-chan quic.Connection) {
	wg := sync.WaitGroup{}
	defer wg.Wait()
	for {
		select {
		case <-ctx.Done():
			return
		case conn, ok := <-connections:
			if !ok {
				return
			}
			sess, err := server.createSession(ctx, conn)
			if err != nil {
				server.logger.Error("create session error ", "remote", conn.RemoteAddr(), "err", err)
				if err = conn.CloseWithError(quic.ApplicationErrorCode(silly_ctrl.UnknownError.Code()), err.Error()); err != nil {
					server.logger.Error("close connection error", "remote", conn.RemoteAddr(), "err", err)
				}
				continue
			}
			wg.Add(1)
			go server.runSession(ctx, sess, &wg)
		}
	}
}
func (server *ctrlNode) runSession(ctx context.Context, sess *session, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		server.logger.Debug("delete session", "id", sess.ID())
		if err := server.manager.Del(sess.ID()); err != nil {
			server.logger.Error("delete session error", "id", sess.ID(), "err", err)
		}
	}()
	if err := sess.run(ctx); err != nil {
		server.logger.Error("run session error", "id", sess.ID(), "err", err)
	}
}
func (server *ctrlNode) createSession(ctx context.Context, conn quic.Connection) (*session, error) {
	app, _, err := server.handshake(ctx, conn)
	if err != nil {
		return nil, err
	}
	server.logger.Info("handshake success", "app", app.AccessKey, "addr", conn.RemoteAddr())
	if _, ok := server.manager.Get(app.AccessKey); ok {
		return nil, silly_ctrl.SessionAlreadyExists
	}
	sess := &session{
		app:           app,
		logger:        server.logger,
		conn:          conn,
		isRemote:      false,
		cfg:           server.cfg,
		handleMapping: server.serviceMapping,
		manager:       server.manager,
	}
	return sess, server.manager.Put(sess)
}
func (server *ctrlNode) handshake(ctx context.Context, conn quic.Connection) (*silly_ctrl.App, *packet.Handshake, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	authStream, err := conn.AcceptStream(ctx)
	if err != nil {
		return nil, nil, err
	}
	go func() {
		<-ctx.Done()
		authStream.CancelRead(quic.StreamErrorCode(silly_ctrl.NoError))
		if err := authStream.Close(); err != nil {
			server.logger.Error("close auth stream error", "err", err)
		}
	}()
	if err = authStream.SetReadDeadline(time.Now().Add(time.Second * server.cfg.HandshakeTimeout)); err != nil {
		return nil, nil, silly_ctrl.AuthError
	}
	hs := &packet.Handshake{}
	if err = protodelim.UnmarshalFrom(packet.NewProtoReader(authStream), hs); err != nil {
		return nil, nil, err
	}
	app, err := server.valid.Validate(hs)
	if err != nil {
		return nil, nil, err
	}
	_, err = protodelim.MarshalTo(authStream, &packet.Ret{
		ErrNo: silly_ctrl.NoError.Code(),
		Msg:   silly_ctrl.NoError.String(),
	})

	if err != nil {
		server.logger.Warn("write handshake ret failed", "err", err)
	}
	return app, hs, err
}
func (server *ctrlNode) Connect(ctx context.Context, addr string, app *silly_ctrl.App, config *tls.Config) error {
	remoteAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := server.tr.Dial(ctx, remoteAddr, config, &server.quicConfig)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.CloseWithError(quic.ApplicationErrorCode(silly_ctrl.ApplicationOver), silly_ctrl.ApplicationOver.Error())
	}()
	err = silly_ctrl.DoQUICRequest[*packet.Handshake, *packet.Ret](
		ctx, app.Signature(), &packet.Ret{}, conn,
		func(ctx context.Context, response *packet.Ret, stream quic.Stream) error {
			if response.ErrNo != silly_ctrl.NoError.Code() {
				return silly_ctrl.ErrorNo(response.ErrNo)
			}
			return err
		})
	if err != nil {
		return err
	}
	sess := &session{
		app:           app,
		logger:        server.logger,
		conn:          conn,
		isRemote:      true,
		handleMapping: server.serviceMapping,
		manager:       server.manager,
		cfg:           server.cfg,
	}

	if err = server.manager.Put(sess); err != nil {
		return err
	}
	return sess.run(ctx)
}
func (server *ctrlNode) Manager() silly_ctrl.SessionManager {
	return server.manager
}
