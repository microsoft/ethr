package client

import (
	"context"
	"fmt"
	"net"
	"time"

	"weavelab.xyz/ethr/client/icmp"
	"weavelab.xyz/ethr/client/tcp"
	"weavelab.xyz/ethr/client/tools"
	"weavelab.xyz/ethr/client/udp"
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/stats"
)

//alias to avoid naming collision on 'Tests'
//type TCPTests = tcp.Tests
//type ICMPTests = icmp.Tests
//type UPDTests = udp.Tests

type Client struct {
	TCPTests  tcp.Tests
	ICMPTests icmp.Tests
	UDPTests  udp.Tests

	NetTools *tools.Tools

	Params ethr.ClientParams
	Logger ethr.Logger
}

func NewClient(isExternal bool, logger ethr.Logger, params ethr.ClientParams, rIP net.IP, rPort uint16, localIP net.IP, localPort uint16) (*Client, error) {
	tools, err := tools.NewTools(isExternal, rIP, rPort, localPort, localIP, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initial network tools: %w", err)
	}

	return &Client{
		NetTools:  tools,
		TCPTests:  tcp.Tests{NetTools: tools, Logger: logger},
		UDPTests:  udp.Tests{NetTools: tools},
		ICMPTests: icmp.Tests{NetTools: tools, Logger: logger},
		Params:    params,
		Logger:    logger,
	}, nil
}

func (c Client) CreateTest(protocol ethr.Protocol, tt ethr.TestType) (*session.Test, error) {
	var aggregator session.ResultAggregator
	if protocol == ethr.TCP {
		switch tt {
		case ethr.TestTypeBandwidth:
			aggregator = tcp.BandwidthAggregator
		case ethr.TestTypeConnectionsPerSecond:
			aggregator = tcp.ConnectionsAggregator
		case ethr.TestTypeLatency:
			aggregator = tcp.LatencyAggregator
		case ethr.TestTypePing:
			aggregator = tcp.PingAggregator
		default:
			// no aggregator for traceroute (single result w/ pointer updates for mtr)
		}
	} else if protocol == ethr.UDP {
		if tt == ethr.TestTypeBandwidth || tt == ethr.TestTypePacketsPerSecond {
			aggregator = udp.BandwidthAggregator
		}
	} else if protocol == ethr.ICMP {
		if tt == ethr.TestTypePing {
			aggregator = icmp.PingAggregator
		}

	}

	c.Logger.Info("Using destination: %s, port: %d", c.NetTools.RemoteIP, c.NetTools.RemotePort)
	publishInterval := time.Second
	if tt == ethr.TestTypePing {
		publishInterval = c.Params.Duration
	}
	test, _ := session.CreateOrGetTest(c.NetTools.RemoteIP, c.NetTools.RemotePort, protocol, tt, c.Params, aggregator, publishInterval)
	test.ClientParam = c.Params
	return test, nil
}

func (c Client) RunTest(ctx context.Context, test *session.Test) error {
	defer close(test.Results)
	stats.StartTimer()
	gap := test.ClientParam.Gap
	test.IsActive = true

	if test.ID.Protocol == ethr.TCP {
		switch test.ID.Type {
		case ethr.TestTypeBandwidth:
			go c.TCPTests.TestBandwidth(test)
		case ethr.TestTypeLatency:
			go c.TCPTests.TestLatency(test, gap)
		case ethr.TestTypeConnectionsPerSecond:
			go c.TCPTests.TestConnectionsPerSecond(test)
		case ethr.TestTypePing:
			go c.TCPTests.TestPing(test, gap, test.ClientParam.WarmupCount)
		case ethr.TestTypeTraceRoute:
			if !c.NetTools.IsAdmin() {
				return fmt.Errorf("must be admin to run traceroute: %w", ErrPermission)
			}
			go c.TCPTests.TestTraceRoute(test, gap, false, 30) // normal traceroute defaults to 64
		case ethr.TestTypeMyTraceRoute:
			if !c.NetTools.IsAdmin() {
				return fmt.Errorf("must be admin to run mytraceroute: %w", ErrPermission)
			}
			go c.TCPTests.TestTraceRoute(test, gap, true, 30) // normal traceroute defaults to 64
		default:
			return ErrNotImplemented
		}
	} else if test.ID.Protocol == ethr.UDP {
		switch test.ID.Type {
		case ethr.TestTypePacketsPerSecond:
			fallthrough
		case ethr.TestTypeBandwidth:
			c.UDPTests.TestBandwidth(test)
		default:
			return ErrNotImplemented
		}
	} else if test.ID.Protocol == ethr.ICMP {
		if !c.NetTools.IsAdmin() {
			return fmt.Errorf("must be admin to run icmp tests: %w", ErrPermission)
		}

		switch test.ID.Type {
		case ethr.TestTypePing:
			go c.ICMPTests.TestPing(test, gap, test.ClientParam.WarmupCount)
		case ethr.TestTypeTraceRoute:
			go c.ICMPTests.TestTraceRoute(test, gap, false, 16) // normal traceroute defaults to 64
		case ethr.TestTypeMyTraceRoute:
			go c.ICMPTests.TestTraceRoute(test, gap, true, 16) // normal traceroute defaults to 64
		default:
			return ErrNotImplemented
		}
	} else {
		return ErrNotImplemented
	}

	//backwards compat with Duration param
	testComplete := time.After(test.ClientParam.Duration)
	select {
	case <-testComplete:
		stats.StopTimer()
		test.Terminate()
		if test.ID.Type == ethr.TestTypePing {
			time.Sleep(500 * time.Millisecond)
		}

		return nil
	case <-test.Done:
		stats.StopTimer()
		time.Sleep(50 * time.Millisecond)
		return nil
	case <-ctx.Done():
		stats.StopTimer()
		test.Terminate()
		if test.ID.Type == ethr.TestTypePing {
			time.Sleep(500 * time.Millisecond)
		}

		return nil
	}
}
