package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)
	defer cancel()
	net.ResolveUDPAddr("udp", "127.0.0.1:65531")
	<-ctx.Done()
}
