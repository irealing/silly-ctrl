package internal

import (
	"github.com/irealing/silly-ctrl"
	"sync"
)

type sessionManager struct {
	rw      sync.RWMutex
	mapping map[string]silly_ctrl.Session
}

func NewManager() silly_ctrl.SessionManager {
	return &sessionManager{mapping: make(map[string]silly_ctrl.Session)}
}

func (manager *sessionManager) Put(sess silly_ctrl.Session) error {
	manager.rw.Lock()
	defer manager.rw.Unlock()
	_, ok := manager.mapping[sess.ID()]
	if ok {
		return silly_ctrl.SessionAlreadyExists
	}
	manager.mapping[sess.ID()] = sess
	return nil
}

func (manager *sessionManager) Get(accessKey string) (silly_ctrl.Session, bool) {
	manager.rw.RLock()
	defer manager.rw.RUnlock()
	sess, ok := manager.mapping[accessKey]
	return sess, ok
}

func (manager *sessionManager) Del(accessKey string) error {
	manager.rw.Lock()
	defer manager.rw.Unlock()
	delete(manager.mapping, accessKey)
	return nil
}
