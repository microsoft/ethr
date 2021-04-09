package client

import (
	"context"
	"fmt"
	"net"
	"strconv"
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

	Params  ethr.ClientParams
	Logger  ethr.Logger
	Session session.Session
}

type TestResult struct {
	Success bool
	Error   error
	Body    interface{}
}

func NewClient(isExternal bool, logger ethr.Logger, session session.Session, params ethr.ClientParams, remote string, localPort uint16, localIP net.IP) (*Client, error) {
	tools, err := tools.NewTools(isExternal, session, remote, localPort, localIP)
	if err != nil {
		return nil, fmt.Errorf("failed to initial network tools: %w", err)
	}

	return &Client{
		NetTools: tools,
		TCPTests: tcp.Tests{NetTools: tools},
		Params:   params,
		Logger:   logger,
		Session:  session,
	}, nil
}

func (c Client) CreateTest(testID session.TestID) (*session.Test, error) {
	if c.NetTools.IsExternal {
		if testID.Protocol != ethr.ICMP && c.NetTools.RemotePort == 0 {
			return nil, fmt.Errorf("in external mode, port cannot be empty for TCP tests")
		}
	} else {
		if c.NetTools.RemotePort != 0 {
			return nil, fmt.Errorf("in client mode, port (%s) cannot be specified in destination (%s)", c.NetTools.RemotePort, c.NetTools.RemoteRaw)
		}
		//port = gEthrPortStr // TODO figure out how to make this less confusing. Why allow a port in 'server' just to force it to the global var?
	}

	c.Logger.Info("Using destination: %s, ip: %s, port: %s", c.NetTools.RemoteHostname, c.NetTools.RemoteIP, c.NetTools.RemotePort)
	test, err := c.Session.NewTest(c.NetTools.RemoteIP.String(), testID, c.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to create new test: %w", err)
	}
	test.RemoteAddr = c.NetTools.RemoteRaw

	test.RemoteIP = c.NetTools.RemoteIP
	test.RemotePort = strconv.Itoa(int(c.NetTools.RemotePort))
	if testID.Protocol == ethr.ICMP {
		test.DialAddr = c.NetTools.RemoteIP.String()
	} else {
		test.DialAddr = fmt.Sprintf("[%s]:%s", c.NetTools.RemoteIP, c.NetTools.RemotePort)
	}
	return test, nil
}

func (c Client) RunTest(ctx context.Context, test *session.Test, results chan TestResult) error {
	stats.StartTimer()
	gap := test.ClientParam.Gap
	test.IsActive = true

	//results := make(chan client.TestResult, 100) // Buffer to allow slop with ui processing (minimum 1 so we don't block trying to send an error and bail)
	if test.ID.Protocol == ethr.TCP {
		switch test.ID.Type {
		case session.TestTypeBandwidth:
			go c.TCPTests.TestBandwidth(test, results)
		case session.TestTypeLatency:
			go c.TestLatency(test, gap, results)
		case session.TestTypeConnectionsPerSecond:
			go c.TestConnectionsPerSecond(test, results)
		case session.TestTypePing:
			go c.TCPTests.TestPing(test, gap, test.ClientParam.WarmupCount, results)
		case session.TestTypeTraceRoute:
			if !c.NetTools.IsAdmin() {
				return fmt.Errorf("must be admin to run traceroute: %w", ErrPermission)
			}
			go c.TCPTests.TestTraceRoute(test, gap, false, 30, results)
		case session.TestTypeMyTraceRoute:
			if !c.NetTools.IsAdmin() {
				return fmt.Errorf("must be admin to run mytraceroute: %w", ErrPermission)
			}
			go c.TCPTests.TestTraceRoute(test, gap, true, 30, results)
		default:
			return ErrNotImplemented
		}
	} else if test.ID.Protocol == ethr.UDP {
		switch test.ID.Type {
		case session.TestTypePacketsPerSecond:
			fallthrough
		case session.TestTypeBandwidth:
			c.UPDTests.TestBandwidth(test, results)
		default:
			return ErrNotImplemented
		}
	} else if test.ID.Protocol == ethr.ICMP {
		if !c.NetTools.IsAdmin() {
			return fmt.Errorf("must be admin to run icmp tests: %w", ErrPermission)
		}

		switch test.ID.Type {
		case session.TestTypePing:
			go c.ICMPTests.TestPing(test, gap, test.ClientParam.WarmupCount, results)
		case session.TestTypeTraceRoute:
			go c.ICMPTests.TestTraceRoute(test, gap, false, 30, results)
		case session.TestTypeMyTraceRoute:
			go c.ICMPTests.TestTraceRoute(test, gap, true, 30, results)
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

		if test.ID.Type == session.TestTypePing {
			time.Sleep(2 * time.Second)
		}

		return nil
	case <-ctx.Done():
		stats.StopTimer()
		close(test.Done)

		test.IsActive = false

		if test.ID.Type == session.TestTypePing {
			time.Sleep(2 * time.Second)
		}

		return nil
	}
}
