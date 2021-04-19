package tcp

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"weavelab.xyz/ethr/session"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/server"
)

func Serve(ctx context.Context, cfg *server.Config, h Handler) error {
	var addr string
	if cfg.LocalIP != nil {
		addr = fmt.Sprintf("%s:%d", cfg.LocalIP, cfg.LocalPort)
	} else {
		addr = fmt.Sprintf(":%d", cfg.LocalPort) // listen on localhost for IPv4 AND IPv6 :/
	}
	l, err := net.Listen(ethr.TCPVersion(cfg.IPVersion), addr)
	if err != nil {
		return err
	}
	defer l.Close()

	// https://golang.org/src/net/http/server.go?s=99574:99629#L3152
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		// TODO break on ctx cancel
		conn, err := l.Accept()
		// If Temporary try again... otherwise bail
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {

				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
				//srv.logf("http: Accept error: %v; retrying in %v", err, tempDelay)
			}

			return err
		}
		conn.RemoteAddr()
		remote, port, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			h.logger.Error("RemoteAddr: Split host port failed: %v", err)
			continue
		}
		rIP := net.ParseIP(remote)
		rPort, _ := strconv.Atoi(port)
		test, _ := session.CreateOrGetTest(rIP, uint16(rPort), ethr.TCP, ethr.TestTypeServer, ServerAggregator)
		if test == nil {
			continue
		}
		go h.HandleConn(ctx, test, conn)
	}
}
