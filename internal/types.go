package internal

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/user"
	"runtime"
	"time"
)

type Worker interface {
	Tag() string
	Run(ctx context.Context) error
}

type WorkerCreator func(ctx context.Context) (Worker, error)

type Heartbeat struct {
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
	UserID    string `json:"user_id"`
	OSName    string `json:"os_name"`
	OSArch    string `json:"os_arch"`
	LocalTime int64  `json:"local_time"`
}

func NewHeartbeat() (*Heartbeat, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	username, err := user.Current()
	if err != nil {
		return nil, err
	}
	return &Heartbeat{
		Hostname:  hostname,
		Username:  username.Name,
		UserID:    username.Username,
		OSName:    runtime.GOOS,
		OSArch:    runtime.GOARCH,
		LocalTime: time.Now().Unix(),
	}, nil
}

type ServiceMeth uint32

const (
	ServiceMethEcho ServiceMeth = iota
	ServiceMethExec
)

type Service interface {
	Meth() ServiceMeth
	Exec(reader io.Reader) (io.Reader, error)
}

type ServiceMapping map[ServiceMeth]Service

func (m ServiceMapping) Register(services ...Service) ServiceMapping {
	for _, service := range services {
		m[service.Meth()] = service
	}
	return m
}
func (m ServiceMapping) Exec(reader io.Reader) (io.Reader, error) {
	buf := make([]byte, 2)
	n, err := reader.Read(buf)
	if err != nil || n != 2 {
		return nil, fmt.Errorf("read meth error %w", err)
	}
	meth := ServiceMeth(binary.LittleEndian.Uint16(buf))
	if service, ok := m[meth]; !ok {
		return nil, err
	} else {
		if ret, err := service.Exec(reader); err != nil {
			return nil, fmt.Errorf("exec %d service %w", service.Meth(), err)
		} else {
			return ret, err
		}
	}
}
