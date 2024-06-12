package impl

import (
	sillyctrl "github.com/irealing/silly-ctrl"
	"github.com/irealing/silly-ctrl/internal"
	"log/slog"
)

func DefaultServices() sillyctrl.ServiceMapping {
	return internal.CreateServiceMapping()
}
func CreateNode(logger *slog.Logger, cfg *sillyctrl.Config, valid sillyctrl.Validator, services sillyctrl.ServiceMapping) (sillyctrl.Node, error) {
	return internal.CreateNode(logger, cfg, valid, services)
}
