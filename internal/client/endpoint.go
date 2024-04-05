package client

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/irealing/silly-ctrl/internal"
	"log/slog"
	"time"
)

var RemoteAddr = "ws://127.0.0.1:8000/ctrl/wsx"

type Endpoint struct {
	services internal.ServiceMapping
	logger   *slog.Logger
}

func (e *Endpoint) Tag() string {
	return "client"
}

func (e *Endpoint) Run(ctx context.Context) error {
	client, err := internal.Dial(ctx, RemoteAddr, nil, e.logger, time.Second*10)
	if err != nil {
		e.logger.Warn("dial ws url error", "url", RemoteAddr, "err", err)
		return err
	}
	return e.runLoop(ctx, client)
}
func (e *Endpoint) runLoop(ctx context.Context, client *internal.WsConn) error {
	writers := make(chan internal.WsReader, 100)
	readers := client.Start(ctx, writers)
	ticker := time.NewTicker(client.TTL)
	defer ticker.Stop()
	for {
		select {
		case r, ok := <-readers:
			if !ok {
				e.logger.Warn("readers closed")
				return nil
			}
			w, err := e.services.Exec(r)
			if err != nil {
				e.logger.Warn("service exec error", "err", err)
			}
			writers <- internal.NewWsReader(websocket.BinaryMessage, w)
		case <-ticker.C:
			beat, err := internal.NewHeartbeat()
			if err != nil {
				e.logger.Warn("service heartbeat error", "err", err)
				return err
			}
			r, err := internal.JsonReader(beat)
			if err != nil {
				e.logger.Warn("make json reader failed", "err", err)
			}
			writers <- internal.NewWsReader(websocket.PingMessage, r)
		}
	}
}
