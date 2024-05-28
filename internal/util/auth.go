package util

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/irealing/silly-ctrl/internal/util/packet"
	"sort"
	"strings"
	"time"
)

type App struct {
	AccessKey string
	Secret    string
}

func (app *App) Signature() *packet.Handshake {
	return GenerateAuthToken(app.AccessKey, app.Secret)
}
func (app *App) Validate(handshake *packet.Handshake) error {
	delay := time.Now().Unix() - int64(handshake.T)
	if delay > 30 || delay < (-30) {
		return SignatureTimeoutError
	}
	if GenerateSignatureString(app.AccessKey, app.Secret, fmt.Sprintf("%d", handshake.T)) != handshake.Sign {
		return HandshakeFailedError
	}
	return nil
}
func GenerateSignature(args ...string) []byte {
	sort.Strings(args)
	val := strings.Join(args, "")
	h := sha256.New()
	h.Write([]byte(val))
	return h.Sum(nil)
}

func GenerateSignatureString(args ...string) string {
	return hex.EncodeToString(GenerateSignature(args...))
}

func GenerateAuthToken(ak, sk string) *packet.Handshake {
	t := time.Now().Unix()
	sign := GenerateSignatureString(ak, sk, fmt.Sprintf("%d", t))
	return &packet.Handshake{
		AccessKey: ak,
		Sign:      sign,
		T:         uint64(t),
	}
}

type Validator interface {
	Validate(handshake *packet.Handshake) (*App, error)
}
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
