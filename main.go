package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/configcat/configcat-proxy/cache"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/grpc"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/web"
)

const (
	exitOk = iota
	exitFailure
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	os.Exit(run(sigChan))
}

func run(closeSignal chan os.Signal) int {
	logger := log.NewLogger(os.Stderr, os.Stdout, log.Warn)
	logger.Reportf("ConfigCat Proxy v%s starting...", sdk.Version())
	var configFile string
	flag.StringVar(&configFile, "c", "", "path to the configuration file")
	flag.Parse()

	conf, err := config.LoadConfigFromFileAndEnvironment(configFile)
	if err != nil {
		logger.Errorf("%s", err)
		return exitFailure
	}
	err = conf.Validate()
	if err != nil {
		logger.Errorf("%s", err)
		return exitFailure
	}

	logger = logger.WithLevel(conf.Log.GetLevel())

	errorChan := make(chan error)
	shutdownFuncs := make([]func(), 0)

	// in the future we might implement an evaluation statistics reporter
	// var evalReporter statistics.Reporter

	statusReporter := status.NewReporter(&conf.Cache)
	telemetryReporter := telemetry.NewReporter(&conf.Diag, sdk.Version(), logger)
	shutdownFuncs = append(shutdownFuncs, func() { telemetryReporter.Shutdown() })

	var diagServer *diag.Server
	if conf.Diag.ShouldRunDiagServer() {
		diagServer = diag.NewServer(&conf.Diag, telemetryReporter, statusReporter, logger, errorChan)
		diagServer.Listen()
		shutdownFuncs = append(shutdownFuncs, func() { diagServer.Shutdown() })
	}

	var externalCache cache.External
	if conf.Cache.IsSet() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // give 15 sec to spin up the cache connection
		defer cancel()

		externalCache, err = cache.SetupExternalCache(ctx, &conf.Cache, telemetryReporter, logger)
		if err != nil {
			return exitFailure
		}
		shutdownFuncs = append(shutdownFuncs, func() { externalCache.Shutdown() })
	}

	sdkRegistrar, err := sdk.NewRegistrar(&conf, telemetryReporter, statusReporter, externalCache, logger)
	if err != nil {
		return exitFailure
	}

	var httpServer *web.Server
	var router *web.HttpRouter
	if conf.Http.Enabled {
		router = web.NewRouter(sdkRegistrar, telemetryReporter, statusReporter, &conf.Http, &conf.Profile, logger)
		httpServer, err = web.NewServer(router, logger, &conf, errorChan)
		if err != nil {
			return exitFailure
		}
		httpServer.Listen()
		shutdownFuncs = append(shutdownFuncs, func() { httpServer.Shutdown() })
	}

	var grpcServer *grpc.Server
	if conf.Grpc.Enabled {
		grpcServer, err = grpc.NewServer(sdkRegistrar, telemetryReporter, statusReporter, &conf, logger, errorChan)
		if err != nil {
			return exitFailure
		}
		grpcServer.Listen()
		shutdownFuncs = append(shutdownFuncs, func() { grpcServer.Shutdown() })
	}

	for {
		select {
		case <-closeSignal:
			logger.Reportf("shutdown requested...")
			sdkRegistrar.Close()

			if router != nil {
				router.Close()
			}

			wg := sync.WaitGroup{}
			wg.Add(len(shutdownFuncs))
			for _, fn := range shutdownFuncs {
				go func(f func()) {
					f()
					wg.Done()
				}(fn)
			}
			wg.Wait()
			return exitOk
		case err = <-errorChan:
			logger.Errorf("%s", err)
			return exitFailure
		}
	}
}
