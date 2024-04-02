package client

import (
	"github.com/irealing/silly-ctrl/internal"
	"io"
)

type EchoService struct {
}

func (EchoService) Meth() internal.ServiceMeth {
	return internal.ServiceMethEcho
}
func (e EchoService) Exec(reader io.Reader) (io.Reader, error) {
	return reader, nil
}
