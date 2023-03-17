package main

import (
	"flag"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/grpc"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/sdk/statistics"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/configcat-proxy/web"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	logger := log.NewLogger(os.Stderr, os.Stdout, log.Warn)
	logger.Reportf("service starting...")
	var configFile string
	flag.StringVar(&configFile, "c", "", "path to the configuration file")
	flag.Parse()

	conf, err := config.LoadConfigFromFileAndEnvironment(configFile)
	if err != nil {
		logger.Errorf("%s", err)
		os.Exit(1)
	}
	err = conf.Validate()
	if err != nil {
		logger.Errorf("%s", err)
		os.Exit(1)
	}

	logger = logger.WithLevel(conf.Log.GetLevel())

	errorChan := make(chan error)

	var metricsHandler metrics.Handler
	var metricServer *metrics.Server
	if conf.Metrics.Enabled {
		metricsHandler = metrics.NewHandler()
		metricServer = metrics.NewServer(metricsHandler.HttpHandler(), &conf.Metrics, logger, errorChan)
		metricServer.Listen()
	}

	var evalReporter statistics.Reporter
	if conf.EvalStats.InfluxDb.Enabled {
		evalReporter = statistics.NewInfluxDbReporter(&conf.EvalStats.InfluxDb)
	}

	statusReporter := status.NewReporter(&conf)
	sdkClients := make(map[string]sdk.Client)
	for key, env := range conf.Environments {
		sdkClients[key] = sdk.NewClient(&sdk.Context{
			SDKConf:        env,
			EvalReporter:   evalReporter,
			MetricsHandler: metricsHandler,
			StatusReporter: statusReporter,
			ProxyConf:      &conf.HttpProxy,
			CacheConf:      &conf.Cache,
			EnvId:          key,
		}, logger)
	}
	router := web.NewRouter(sdkClients, metricsHandler, statusReporter, &conf.Http, logger)

	httpServer := web.NewServer(router.Handler(), logger, &conf, errorChan)
	httpServer.Listen()

	var grpcServer *grpc.Server
	if conf.Grpc.Enabled {
		grpcServer = grpc.NewServer(sdkClients, metricsHandler, &conf, logger, errorChan)
		grpcServer.Listen()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	for {
		select {
		case <-sigChan:
			if evalReporter != nil {
				evalReporter.Close()
			}
			for _, sdkClient := range sdkClients {
				sdkClient.Close()
			}
			router.Close()

			shutDownCount := 1
			if metricServer != nil {
				shutDownCount++
			}
			if grpcServer != nil {
				shutDownCount++
			}

			wg := sync.WaitGroup{}
			wg.Add(shutDownCount)
			go func() {
				httpServer.Shutdown()
				wg.Done()
			}()
			if grpcServer != nil {
				go func() {
					grpcServer.Shutdown()
					wg.Done()
				}()
			}
			if metricServer != nil {
				go func() {
					metricServer.Shutdown()
					wg.Done()
				}()
			}
			wg.Wait()
			os.Exit(0)
		case err = <-errorChan:
			logger.Errorf("%s", err)
			os.Exit(1)
		}
	}
}
