package impl

import "github.com/irealing/silly-ctrl/internal/ctrl"

func createServiceMapping() ctrl.ServiceMapping {
	ret := make(ctrl.ServiceMapping)
	return ret.Register(forwardService{})
}
