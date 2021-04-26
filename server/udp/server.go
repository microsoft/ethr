package udp

import (
	"context"
	"fmt"
	"net"
	"runtime"

	"weavelab.xyz/ethr/config"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/server"
)

func Serve(ctx context.Context, cfg *server.Config, h Handler) error {
	addr := config.GetAddrString(cfg.LocalIP, cfg.LocalPort)
	udpAddr, err := net.ResolveUDPAddr(ethr.UDPVersion(cfg.IPVersion), addr)
	if err != nil {
		return fmt.Errorf("unable to resolve UDP address: %w", err)
	}
	l, err := net.ListenUDP(ethr.UDPVersion(cfg.IPVersion), udpAddr)
	if err != nil {
		return fmt.Errorf("error listening on %s for UDP pkt/s tests: %w", cfg.LocalPort, err)
	}

	// Set socket buffer to 4MB per CPU so we can queue 4MB per CPU in case Ethr is not
	// able to keep up temporarily.
	// TODO figure out why this doesn't work sometimes
	_ = l.SetReadBuffer(runtime.NumCPU() * 4 * 1024 * 1024)
	//if err != nil {
	//	return fmt.Errorf("failed to set ReadBuffer on UDP socket: %w", err)
	//}

	//
	// We use NumCPU here instead of NumThreads passed from client. The
	// reason is that for UDP, there is no connection, so all packets come
	// on same CPU, so it isn't clear if there are any benefits to running
	// more threads than NumCPU(). TODO: Evaluate this in future.
	for i := 0; i < runtime.NumCPU(); i++ {
		go h.HandleConn(ctx, nil, l)
	}
	return nil
}
