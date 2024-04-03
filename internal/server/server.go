package server

import (
	"github.com/gorilla/websocket"
	"github.com/irealing/silly-ctrl/internal"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type EndpointManager struct {
	rw      sync.RWMutex
	mapping map[uint64]*Session
	upgrade *websocket.Upgrader
	logger  *slog.Logger
}

func (manager *EndpointManager) Accept(w http.ResponseWriter, r *http.Request) (*Session, error) {
	conn, err := manager.upgrade.Upgrade(w, r, nil)
	if err != nil {
		manager.logger.Warn("accept error", "err", err)
		return nil, err
	}
	wsConn := internal.NewWSClient(conn, manager.logger, time.Second*15)
	sess := NewSession(wsConn, manager.logger)
	return sess, nil
}
func (manager *EndpointManager) runSessionLoop(session *Session) {
	//	TODO: sessionLoop
}
func (m *EndpointManager) putSession(session *Session) {
	m.rw.Lock()
	defer m.rw.Unlock()
	m.mapping[session.conn.ID] = session
}
func (m *EndpointManager) getSession(id uint64) (*Session, bool) {
	m.rw.RLock()
	defer m.rw.RUnlock()
	sess, ok := m.mapping[id]
	return sess, ok
}
func (m *EndpointManager) delSession(id uint64) {
	m.rw.Lock()
	defer m.rw.RUnlock()
	delete(m.mapping, id)
}
