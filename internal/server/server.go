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

func (manager *EndpointManager) Accept(w http.ResponseWriter, r *http.Request) (*internal.WsConn, error) {
	conn, err := manager.upgrade.Upgrade(w, r, nil)
	if err != nil {
		manager.logger.Warn("accept error", "err", err)
		return nil, err
	}
	return internal.NewWSClient(conn, manager.logger, time.Second*15), err
}
