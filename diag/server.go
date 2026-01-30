package diag

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
)

type Server struct {
	httpServer   *http.Server
	log          log.Logger
	conf         *config.DiagConfig
	errorChannel chan error
}

func NewServer(conf *config.DiagConfig, telemetryReporter telemetry.Reporter, statusReporter status.Reporter, log log.Logger, errorChan chan error) *Server {
	diagLog := log.WithPrefix("diag")
	mux := http.NewServeMux()

	if conf.IsMetricsEnabled() && conf.IsPrometheusExporterEnabled() {
		mux.Handle("/metrics", telemetryReporter.GetPrometheusHttpHandler())
	}

	if conf.IsStatusEnabled() {
		mux.Handle("/status", statusReporter.HttpHandler())
		diagLog.Reportf("status enabled, accepting requests on path: /status")
	}

	setupDebugEndpoints(mux)

	httpServer := &http.Server{
		Addr:         ":" + strconv.Itoa(conf.Port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{
		log:          diagLog,
		httpServer:   httpServer,
		conf:         conf,
		errorChannel: errorChan,
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
