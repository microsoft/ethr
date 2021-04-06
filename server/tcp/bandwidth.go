package tcp

import (
	"fmt"
	"net"
	"sync/atomic"
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/stats"
)

func (h Handler) TestBandwidth(test *session.Test, clientParam ethr.ClientParams, conn net.Conn) (*session.BandwidthResult, error){
	size := clientParam.BufferSize
	buff := make([]byte, size)
	for i := uint32(0); i < size; i++ {
		buff[i] = byte(i)
	}
	bufferLen := len(buff)
	totalBytesToSend := test.ClientParam.BwRate
	sentBytes := uint64(0)
	start, waitTime, bytesToSend := stats.BeginThrottle(totalBytesToSend, bufferLen)
	for {
		n := 0
		var err error
		if clientParam.Reverse {
			n, err = conn.Write(buff[:bytesToSend])
		} else {
			n, err = conn.Read(buff)
		}
		if err != nil {
			return nil, fmt.Errorf("error sending/receiving data on a connection for bandwidth test: %w", err)
		}
		atomic.AddUint64(&test.Result.Bandwidth, uint64(size))
		if clientParam.Reverse {
			sentBytes += uint64(n)
			start, waitTime, sentBytes, bytesToSend = stats.EnforceThrottle(start, waitTime, totalBytesToSend, sentBytes, bufferLen)
		}
	}
}
