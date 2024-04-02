package server

import (
	"context"
	"fmt"
	"github.com/irealing/silly-ctrl/internal"
	"log/slog"
	"sync"
	"sync/atomic"
)

type Session struct {
	conn      *internal.WsConn
	r         <-chan internal.WsReader
	w         chan internal.WsReader
	logger    *slog.Logger
	heartbeat *internal.Heartbeat
	lock      sync.RWMutex
	started   int32
}

func NewSession(conn *internal.WsConn, logger *slog.Logger, heartbeat *internal.Heartbeat) *Session {
	//w := make(chan internal.WsReader, 10)
	return &Session{conn: conn, logger: logger, heartbeat: heartbeat}
}
func (s *Session) Run(ctx context.Context) {
	atomic.StoreInt32(&s.started, 1)
	defer atomic.StoreInt32(&s.started, 0)
	s.runIt(ctx)
}
func (s *Session) markStatus(val int32) {
	defer s.lock.Unlock()
	s.lock.Lock()
}
func (s *Session) runIt(ctx context.Context) {
	s.w = make(chan internal.WsReader, 10)
	s.r = s.conn.Run(ctx, s.w)
}
func (s *Session) Exec(reader internal.WsReader) (internal.WsReader, error) {
	if atomic.LoadInt32(&s.started) > 0 {
		return nil, fmt.Errorf("session not started")
	}
	return nil, nil
}
