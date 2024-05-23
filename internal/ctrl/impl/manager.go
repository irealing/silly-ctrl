package impl

import (
	"github.com/irealing/silly-ctrl/internal/ctrl"
	"github.com/irealing/silly-ctrl/internal/util"
	"sync"
)

type sessionManager struct {
	rw      sync.RWMutex
	mapping map[string]ctrl.Session
}

func NewManager() ctrl.SessionManager {
	return &sessionManager{mapping: make(map[string]ctrl.Session)}
}

func (manager *sessionManager) Put(sess ctrl.Session) error {
	manager.rw.Lock()
	defer manager.rw.Unlock()
	_, ok := manager.mapping[sess.ID()]
	if ok {
		return util.SessionAlreadyExists
	}
	manager.mapping[sess.ID()] = sess
	return nil
}

func (manager *sessionManager) Get(accessKey string) (ctrl.Session, bool) {
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
