package main

import (
	"context"
	"flag"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag"
	"github.com/configcat/configcat-proxy/diag/metrics"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/grpc"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/sdk/store/cache"
	"github.com/configcat/configcat-proxy/web"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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
	logger.Reportf("service starting...")
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

	// in the future we might implement an evaluation statistics reporter
	// var evalReporter statistics.Reporter

	statusReporter := status.NewReporter(&conf)

	var metricsReporter metrics.Reporter
	if conf.Diag.Metrics.Enabled {
		metricsReporter = metrics.NewReporter()
	}

	var diagServer *diag.Server
	if conf.Diag.Enabled && (conf.Diag.Metrics.Enabled || conf.Diag.Status.Enabled) {
		diagServer = diag.NewServer(&conf.Diag, statusReporter, metricsReporter, logger, errorChan)
		diagServer.Listen()
	}

	var externalCache cache.External
	if conf.Cache.IsSet() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // give 5 sec to spin up the cache connection
		defer cancel()

		externalCache, err = cache.SetupExternalCache(ctx, &conf.Cache, logger)
		if err != nil {
			return exitFailure
		}
	}

	sdkClients := make(map[string]sdk.Client)
	for key, sdkConf := range conf.SDKs {
		sdkClients[key] = sdk.NewClient(&sdk.Context{
			SDKConf:            sdkConf,
			EvalReporter:       nil,
			MetricsReporter:    metricsReporter,
			StatusReporter:     statusReporter,
			ProxyConf:          &conf.HttpProxy,
			GlobalDefaultAttrs: conf.DefaultAttrs,
			SdkId:              key,
			ExternalCache:      externalCache,
		}, logger)
	}

	var httpServer *web.Server
	var router *web.HttpRouter
	if conf.Http.Enabled {
		router = web.NewRouter(sdkClients, metricsReporter, statusReporter, &conf.Http, logger)
		httpServer, err = web.NewServer(router.Handler(), logger, &conf, errorChan)
		if err != nil {
			return exitFailure
		}
		httpServer.Listen()
	}

	var grpcServer *grpc.Server
	if conf.Grpc.Enabled {
		grpcServer, err = grpc.NewServer(sdkClients, metricsReporter, statusReporter, &conf, logger, errorChan)
		if err != nil {
			return exitFailure
		}
		grpcServer.Listen()
	}

	for {
		select {
		case <-closeSignal:
			for _, sdkClient := range sdkClients {
				sdkClient.Close()
			}
			if router != nil {
				router.Close()
			}

			shutDownCount := 0
			if externalCache != nil {
				shutDownCount++
			}
			if httpServer != nil {
				shutDownCount++
			}
			if grpcServer != nil {
				shutDownCount++
			}
			if diagServer != nil {
				shutDownCount++
			}
			wg := sync.WaitGroup{}
			wg.Add(shutDownCount)
			if externalCache != nil {
				go func() {
					externalCache.Shutdown()
					wg.Done()
				}()
			}
			if httpServer != nil {
				go func() {
					httpServer.Shutdown()
					wg.Done()
				}()
			}
			if diagServer != nil {
				go func() {
					diagServer.Shutdown()
					wg.Done()
				}()
			}
			if grpcServer != nil {
				go func() {
					grpcServer.Shutdown()
					wg.Done()
				}()
			}
			wg.Wait()
			return exitOk
		case err = <-errorChan:
			logger.Errorf("%s", err)
			return exitFailure
		}
	}
}
