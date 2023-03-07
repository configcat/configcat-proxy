package main

import (
	"flag"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/grpc"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/configcat-proxy/web"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	logger := log.NewLogger(os.Stderr, os.Stdout, log.Warn)
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

	var metric metrics.Handler
	var metricServer *metrics.Server
	if conf.Metrics.Enabled {
		metric = metrics.NewHandler()
		metricServer = metrics.NewServer(metric.HttpHandler(), conf.Metrics, logger, errorChan)
		metricServer.Listen()
	}

	statusReporter := status.NewReporter(conf)
	sdkClient := sdk.NewClient(conf.SDK, conf.HttpProxy, metric, statusReporter, logger)
	router := web.NewRouter(sdkClient, metric, statusReporter, conf.Http, logger)

	httpServer := web.NewServer(router.Handler(), logger, conf, errorChan)
	httpServer.Listen()

	var grpcServer *grpc.Server
	if conf.Grpc.Enabled {
		grpcServer = grpc.NewServer(sdkClient, metric, conf, logger, errorChan)
		grpcServer.Listen()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	for {
		select {
		case <-sigChan:
			sdkClient.Close()
			router.Close()

			shutDownCount := 1
			if metric != nil {
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
			if metric != nil {
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
