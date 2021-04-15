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
type TCPTests = tcp.Tests
type ICMPTests = icmp.Tests
type UPDTests = udp.Tests

type Client struct {
	TCPTests
	ICMPTests
	UPDTests

	NetTools *tools.Tools

	Params ethr.ClientParams
	Logger ethr.Logger
}

func NewClient(isExternal bool, logger ethr.Logger, params ethr.ClientParams, remote string, localIP net.IP, localPort uint16) (*Client, error) {
	tools, err := tools.NewTools(isExternal, remote, localPort, localIP)
	if err != nil {
		return nil, fmt.Errorf("failed to initial network tools: %w", err)
	}

	return &Client{
		NetTools: tools,
		TCPTests: tcp.Tests{NetTools: tools},
		Params:   params,
		Logger:   logger,
	}, nil
}

func (c Client) CreateTest(protocol ethr.Protocol, tt ethr.TestType) (*session.Test, error) {
	if c.NetTools.IsExternal {
		if protocol != ethr.ICMP && c.NetTools.RemotePort == 0 {
			return nil, fmt.Errorf("in external mode, port cannot be empty for TCP tests")
		}
	} else {
		if c.NetTools.RemotePort != 0 {
			return nil, fmt.Errorf("in client mode, port (%s) cannot be specified in destination (%s)", c.NetTools.RemotePort, c.NetTools.RemoteRaw)
		}
		//port = gEthrPortStr // TODO figure out how to make this less confusing. Why allow a port in 'server' just to force it to the global var?
	}

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
		if tt == ethr.TestTypeBandwidth {
			aggregator = udp.BandwidthAggregator
		}
	} else if protocol == ethr.ICMP {
		if tt == ethr.TestTypePing {
			aggregator = icmp.PingAggregator
		}

	}

	c.Logger.Info("Using destination: %s, ip: %s, port: %s", c.NetTools.RemoteHostname, c.NetTools.RemoteIP, c.NetTools.RemotePort)
	test, _ := session.CreateOrGetTest(c.NetTools.RemoteIP, c.NetTools.RemotePort, protocol, tt, aggregator)
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
			go c.TestLatency(test, gap)
		case ethr.TestTypeConnectionsPerSecond:
			go c.TestConnectionsPerSecond(test)
		case ethr.TestTypePing:
			go c.TCPTests.TestPing(test, gap, test.ClientParam.WarmupCount)
		case ethr.TestTypeTraceRoute:
			if !c.NetTools.IsAdmin() {
				return fmt.Errorf("must be admin to run traceroute: %w", ErrPermission)
			}
			go c.TCPTests.TestTraceRoute(test, gap, false, 30)
		case ethr.TestTypeMyTraceRoute:
			if !c.NetTools.IsAdmin() {
				return fmt.Errorf("must be admin to run mytraceroute: %w", ErrPermission)
			}
			go c.TCPTests.TestTraceRoute(test, gap, true, 30)
		default:
			return ErrNotImplemented
		}
	} else if test.ID.Protocol == ethr.UDP {
		switch test.ID.Type {
		case ethr.TestTypePacketsPerSecond:
			fallthrough
		case ethr.TestTypeBandwidth:
			c.UPDTests.TestBandwidth(test)
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
			go c.ICMPTests.TestTraceRoute(test, gap, false, 30)
		case ethr.TestTypeMyTraceRoute:
			go c.ICMPTests.TestTraceRoute(test, gap, true, 30)
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
		close(test.Done)

		test.IsActive = false

		if test.ID.Type == ethr.TestTypePing {
			time.Sleep(2 * time.Second)
		}

		return nil
	case <-ctx.Done():
		stats.StopTimer()
		close(test.Done)

		test.IsActive = false

		if test.ID.Type == ethr.TestTypePing {
			time.Sleep(2 * time.Second)
		}

		return nil
	}
}
