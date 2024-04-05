package server

import (
	"context"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/irealing/silly-ctrl/internal"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Manager struct {
	rw       sync.RWMutex
	mapping  map[uint64]*Session
	upgrade  *websocket.Upgrader
	logger   *slog.Logger
	status   int32
	sessions chan *Session
}

func NewManager(logger *slog.Logger) *Manager {
	return &Manager{logger: logger, mapping: make(map[uint64]*Session), upgrade: &websocket.Upgrader{}}
}

func (manager *Manager) Accept(w http.ResponseWriter, r *http.Request) (*Session, error) {
	if atomic.LoadInt32(&manager.status) != 1 {
		return nil, errors.New("sessions manager bad status")
	}
	conn, err := manager.upgrade.Upgrade(w, r, nil)
	if err != nil {
		manager.logger.Warn("accept error", "err", err)
		return nil, err
	}
	wsConn := internal.NewWSConn(conn, manager.logger, time.Second*15)
	sess := NewSession(wsConn, manager.logger)
	manager.sessions <- sess
	return sess, nil
}
func (manager *Manager) Run(ctx context.Context) {
	manager.sessions = make(chan *Session, 10)
	defer close(manager.sessions)
	atomic.StoreInt32(&manager.status, 1)
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	defer atomic.StoreInt32(&manager.status, 0)
	for {
		select {
		case sess := <-manager.sessions:
			wg.Add(1)
			go manager.runSessionLoop(ctx, sess, wg)
		case <-ctx.Done():
			manager.logger.Warn("session manager stop with context done")
			return
		}
	}
}
func (manager *Manager) runSessionLoop(ctx context.Context, session *Session, wg *sync.WaitGroup) {
	manager.putSession(session)
	defer wg.Done()
	defer manager.delSession(session.conn.ID)
	session.Run(ctx)
}
func (manager *Manager) putSession(session *Session) {
	manager.rw.Lock()
	defer manager.rw.Unlock()
	manager.mapping[session.conn.ID] = session
}
func (manager *Manager) getSession(id uint64) (*Session, bool) {
	manager.rw.RLock()
	defer manager.rw.RUnlock()
	sess, ok := manager.mapping[id]
	return sess, ok
}
func (manager *Manager) delSession(id uint64) {
	manager.rw.Lock()
	defer manager.rw.Unlock()
	delete(manager.mapping, id)
}
