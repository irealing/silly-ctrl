package silly_ctrl

import (
	"github.com/irealing/silly-ctrl/packet"
)

type basicValidator struct {
	apps map[string]App
}

func NewBasicValidator(apps []App) Validator {
	mapping := make(map[string]App)
	for _, app := range apps {
		mapping[app.AccessKey] = app
	}
	return &basicValidator{apps: mapping}
}

func (b *basicValidator) Validate(handshake *packet.Handshake) (*App, error) {
	app, ok := b.apps[handshake.AccessKey]
	if !ok {
		return nil, UnknownAppError
	}
	return &app, app.Validate(handshake)
}
