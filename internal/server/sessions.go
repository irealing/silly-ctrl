package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/irealing/silly-ctrl/internal"
	"log/slog"
	"sync/atomic"
)

const (
	statusInit int32 = iota
	statusRunning
	statusStopped
)

type Session struct {
	conn   *internal.WsConn
	r      chan internal.WsReader
	w      chan internal.WsReader
	logger *slog.Logger
	status int32
	cancel context.CancelFunc
}

func NewSession(conn *internal.WsConn, logger *slog.Logger) *Session {
	return &Session{conn: conn, status: statusInit, logger: logger}
}
func (s *Session) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	if err := s.runIt(ctx); err != nil {
		s.logger.Warn("run session error", "id", s.conn.ID, "err", err)
	}
}

func (s *Session) runIt(ctx context.Context) error {
	s.w = make(chan internal.WsReader, 10)
	s.r = make(chan internal.WsReader, 10)
	atomic.StoreInt32(&s.status, statusRunning)
	defer atomic.StoreInt32(&s.status, statusStopped)
	return s.conn.Run(ctx, s.w, s.r)
}

func (s *Session) Exec(reader internal.WsReader) (internal.WsReader, error) {
	if atomic.LoadInt32(&s.status) != statusRunning {
		return nil, fmt.Errorf("wrong status %d", s.conn.ID)
	}
	s.w <- reader
	ret, ok := <-s.r
	if !ok {
		return nil, errors.New("read on closed channel")
	}
	return ret, nil
}

func (s *Session) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}
