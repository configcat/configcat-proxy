package diag

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/metrics"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/log"
)

type Server struct {
	httpServer      *http.Server
	log             log.Logger
	conf            *config.DiagConfig
	errorChannel    chan error
	statusReporter  status.Reporter
	metricsReporter metrics.Reporter
}

func NewServer(conf *config.DiagConfig, statusReporter status.Reporter, metricsReporter metrics.Reporter, log log.Logger, errorChan chan error) *Server {
	diagLog := log.WithPrefix("diag")
	mux := http.NewServeMux()

	if metricsReporter != nil && conf.Metrics.Enabled {
		mux.Handle("/metrics", metricsReporter.HttpHandler())
		diagLog.Reportf("metrics enabled, accepting requests on path: /metrics")
	}

	if statusReporter != nil && conf.Status.Enabled {
		mux.Handle("/status", statusReporter.HttpHandler())
		diagLog.Reportf("status enabled, accepting requests on path: /status")
	}

	setupDebugEndpoints(mux)

	httpServer := &http.Server{
		Addr:    ":" + strconv.Itoa(conf.Port),
		Handler: mux,
	}

	return &Server{
		log:             diagLog,
		httpServer:      httpServer,
		conf:            conf,
		errorChannel:    errorChan,
		statusReporter:  statusReporter,
		metricsReporter: metricsReporter,
	}
}

func (s *Server) Listen() {
	if s.httpServer == nil {
		return
	}

	s.log.Reportf("diag HTTP server listening on port: %d", s.conf.Port)

	go func() {
		httpErr := s.httpServer.ListenAndServe()

		if !errors.Is(httpErr, http.ErrServerClosed) {
			s.errorChannel <- fmt.Errorf("error starting diag HTTP server on port: %d  %s", s.conf.Port, httpErr)
		}
	}()
}

func (s *Server) Shutdown() {
	if s.httpServer == nil {
		return
	}

	s.log.Reportf("initiating server shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		s.log.Errorf("shutdown error: %s", err)
	}
	s.log.Reportf("server shutdown complete")
}
