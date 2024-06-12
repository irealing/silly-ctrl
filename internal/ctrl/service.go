package ctrl

import (
	"github.com/irealing/silly-ctrl"
)

func createServiceMapping() silly_ctrl.ServiceMapping {
	ret := make(silly_ctrl.ServiceMapping)
	return ret.Register(forwardService{}).
		Register(proxyService{}).
		Register(execService{}).
		Register(emptyService{})
}
