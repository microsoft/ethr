package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"weavelab.xyz/ethr/server/udp"

	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/stats"

	"weavelab.xyz/ethr/log"

	"weavelab.xyz/ethr/client"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/server/tcp"

	"weavelab.xyz/ethr/config"
	"weavelab.xyz/ethr/server"
	cUi "weavelab.xyz/ethr/ui/client"
	serverUi "weavelab.xyz/ethr/ui/server"
)

func main() {
	//
	// Set GOMAXPROCS to 1024 as running large number of goroutines that send
	// data in a tight loop over network is resulting in unfair time allocation
	// across goroutines causing starvation of many TCP connections. Using a
	// higher number of threads via GOMAXPROCS solves this problem.
	//
	runtime.GOMAXPROCS(1024)

	fmt.Println("\nEthr: Comprehensive Network Performance Measurement Tool (Version: " + config.Version + ")")
	fmt.Println("Maintainer: Pankaj Garg (ipankajg @ LinkedIn | GitHub | Gmail | Twitter)")
	fmt.Println("")

	err := config.Init()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Please use \"ethr -h\" for complete list of command line arguments\n")
		os.Exit(1)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("Shutting down...")
		cancel()
	}()

	if config.IsServer {
		cfg := server.Config{
			IPVersion: config.IPVersion,
			LocalIP:   config.LocalIP,
			LocalPort: config.Port,
		}

		term := serverUi.NewUI(config.ShowUI)
		term.Display(ctx)

		logger := configureLogger(ctx, term)

		stats.StartTimer()
		defer stats.StopTimer()

		// TODO log listening ip and port
		// TODO stop server on ctx.cancel
		err := udp.Serve(ctx, &cfg, udp.NewHandler(logger))
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(1)
		}

		err = tcp.Serve(ctx, &cfg, tcp.NewHandler(logger))
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(1)
		}
	} else {
		logger := configureLogger(ctx, nil)
		term := cUi.NewUI(config.Title, !config.NoConnectionStats, logger)
		params := ethr.ClientParams{
			NumThreads:  uint32(config.ThreadCount),
			BufferSize:  uint32(config.BufferSize),
			RttCount:    uint32(config.Iterations),
			Reverse:     config.Reverse,
			Duration:    config.Duration,
			Gap:         config.Gap,
			WarmupCount: uint32(config.WarmupCount),
			BwRate:      config.BandwidthRate,
			ToS:         uint8(config.TOS),
		}
		c, err := client.NewClient(config.IsExternal, logger, params, config.RemoteIP, config.Port, config.LocalIP, config.LocalPort)
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(1)
		}
		test, err := c.CreateTest(config.Protocol, config.TestType)
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(1)
		}

		go term.PrintTestResults(ctx, test)

		err = c.RunTest(ctx, test)
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(1)
		}
	}
}

func configureLogger(ctx context.Context, term *serverUi.UI) ethr.Logger {
	loglevel := log.LevelInfo
	if config.Debug {
		loglevel = log.LevelDebug
	}
	loggers := make([]ethr.Logger, 0)
	if !config.NoOutput {
		fileLogger, err := log.NewJSONLogger(config.OutputFile, loglevel)
		if err != nil {
			fmt.Printf("failed to initialize file logger: %s\n", err)
		}
		fileLogger.Init(ctx)
		loggers = append(loggers, fileLogger)
	}

	if config.IsServer && config.ShowUI {
		termLogger := log.NewTuiLogger(loglevel, term.Terminal)
		termLogger.Init(ctx)
		loggers = append(loggers, termLogger)
	} else {
		stdoutLogger := log.NewSTDOutLogger(loglevel)
		stdoutLogger.Init(ctx)
		loggers = append(loggers, stdoutLogger)
	}
	logger := log.NewAggregateLogger(loggers...)
	session.Logger = logger
	stats.Logger = logger
	return logger
}
