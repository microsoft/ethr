package tcp

import (
	"net"
	"time"
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/server"
)



type HandlerFunc func(net.Conn)

func Serve(cfg *server.Config, h HandlerFunc) error {
	l, err := net.Listen(ethr.TCPVersion(cfg.IPVersion), cfg.LocalIP+":"+cfg.LocalPort)
	if err != nil {
		return err
	}
	defer l.Close()

	// https://golang.org/src/net/http/server.go?s=99574:99629#L3152
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
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
		go h(conn)
	}
}


