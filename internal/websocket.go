package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

var globalClientID uint64 = 0

type WsReader interface {
	io.Reader
	Type() int
}

type simpleReader struct {
	typing int
	reader io.Reader
}

func NewWsReader(typing int, reader io.Reader) WsReader {
	return &simpleReader{typing: typing, reader: reader}
}

func (s *simpleReader) Read(p []byte) (n int, err error) {
	return s.reader.Read(p)
}

func (s *simpleReader) Type() int {
	return s.typing
}
func JsonReader(v any) (io.Reader, error) {
	bf := bytes.NewBuffer(nil)
	return bf, json.NewEncoder(bf).Encode(v)
}

type WsConn struct {
	ID     uint64
	conn   *websocket.Conn
	logger *slog.Logger
	TTL    time.Duration
}

func Dial(ctx context.Context, url string, header http.Header, logger *slog.Logger, ttl time.Duration) (*WsConn, error) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, header)
	if err != nil {
		logger.Warn("dial error", "url", url, "err", err)
		return nil, err
	}
	return &WsConn{ID: atomic.AddUint64(&globalClientID, 1), conn: conn, logger: logger, TTL: ttl}, nil
}

func NewWSClient(conn *websocket.Conn, logger *slog.Logger, ttl time.Duration) *WsConn {
	return &WsConn{conn: conn, logger: logger, TTL: ttl}
}
func (w *WsConn) Start(ctx context.Context, readers <-chan WsReader) <-chan WsReader {
	ret := make(chan WsReader, 10)
	go func() {
		if err := w.Run(ctx, readers, ret); err != nil {
			w.logger.Warn("client Run error", "client", w.ID, "err", err)
		}
	}()
	return ret
}
func (w *WsConn) Run(ctx context.Context, readers <-chan WsReader, ret chan<- WsReader) error {
	ctx, cancel := context.WithCancel(ctx)
	eg, ctx := errgroup.WithContext(ctx)
	go func() {
		defer close(ret)
		if err := eg.Wait(); err != nil {
			w.logger.Warn("client loop error", "error", err)
		}
	}()
	eg.Go(func() error {
		defer cancel()
		return w.readLoop(ctx, ret)
	})
	eg.Go(func() error {
		defer cancel()
		return w.writeLoop(ctx, readers)
	})
	return eg.Wait()
}
func (w *WsConn) readLoop(ctx context.Context, writers chan<- WsReader) error {
	for {
		select {
		case <-ctx.Done():
			w.logger.Warn("client context done")
			return nil
		default:
			err := w.conn.SetReadDeadline(time.Now().Add(w.TTL))
			if err != nil {
				return fmt.Errorf("read timeout")
			}
			messageType, reader, err := w.conn.NextReader()
			if err != nil {
				return err
			}
			writers <- NewWsReader(messageType, reader)
		}
	}
}
func (w *WsConn) writeLoop(ctx context.Context, readers <-chan WsReader) error {
	for {
		select {
		case <-ctx.Done():
			w.logger.Warn("client context done")
			return nil
		case r, ok := <-readers:
			if !ok {
				w.logger.Warn("readers closed")
				return nil
			}
			if err := w.conn.SetWriteDeadline(time.Now().Add(w.TTL)); err != nil {
				return fmt.Errorf("set write deadline error %w", err)
			}
			writer, err := w.conn.NextWriter(r.Type())
			if err != nil {
				return err
			}
			if _, err := io.Copy(writer, r); err != nil {
				w.logger.Warn("write message error", "messageType", r.Type(), "error", err)
				return err
			}
			if err := writer.Close(); err != nil {
				w.logger.Warn("close writer error", "error", err)
				return err
			}
		}
	}
}
