package util

import (
	"context"
	"errors"
	"github.com/irealing/silly-ctrl/internal/util/packet"
	"github.com/quic-go/quic-go"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
	"io"
)

func RetWithError(e error) *packet.Ret {
	r := &packet.Ret{}
	if e == nil {
		return r
	}
	r.Msg = e.Error()
	var errNo ErrorNo
	ok := errors.As(e, &errNo)
	if !ok {
		r.ErrNo = UnknownError.Code()
	} else {
		r.ErrNo = errNo.Code()
	}
	return r
}

func CopyWithContext(ctx context.Context, src io.Reader, dst io.Writer) error {
	buf := make([]byte, 1024*32)
	errDone := make(chan int, 1)
	var (
		err    error
		nr     int
		isOpen bool
	)
	go func() {
		defer close(errDone)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				nr, err = src.Read(buf)
				if err != nil {
					return
				}
				errDone <- nr
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case nr, isOpen = <-errDone:
			if nr > 0 {
				_, err = dst.Write(buf[:nr])
				if err != nil {
					return err
				}
			}
			if !isOpen || err != nil {
				return err
			}
		}
	}
}

func Forward(ctx context.Context, x, y io.ReadWriter) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return CopyWithContext(ctx, x, y)
	})
	eg.Go(func() error {
		return CopyWithContext(ctx, y, x)
	})
	return eg.Wait()
}

type RequestCallback[R proto.Message] func(ctx context.Context, response R, stream quic.Stream) error

func DoQUICRequest[T, R proto.Message](ctx context.Context, msg T, ret R, conn quic.Connection, callback RequestCallback[R]) error {
	stream, err := conn.OpenStream()
	if err != nil {
		return err
	}
	defer func() {
		_ = stream.Close()
	}()
	_, err = protodelim.MarshalTo(stream, msg)
	if err != nil {
		return err
	}
	err = protodelim.UnmarshalFrom(packet.NewProtoReader(stream), ret)
	if err != nil {
		return err
	}
	if callback == nil {
		return err
	}
	return callback(ctx, ret, stream)
}
