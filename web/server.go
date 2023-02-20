package web

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"net/http"
	"time"
)

type Server struct {
	log          log.Logger
	conf         config.Config
	httpServer   *http.Server
	errorChannel chan error
}

func NewServer(handler http.Handler, log log.Logger, conf config.Config, errorChan chan error) *Server {
	httpLog := log.WithPrefix("http")
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", conf.Http.Port),
		Handler: handler,
	}
	if conf.Tls.Enabled {
		t := &tls.Config{
			MinVersion: conf.Tls.GetVersion(),
			ServerName: conf.Tls.ServerName,
		}
		for _, c := range conf.Tls.Certificates {
			if cert, err := tls.LoadX509KeyPair(c.Cert, c.Key); err == nil {
				t.Certificates = append(t.Certificates, cert)
			}
		}
		httpServer.TLSConfig = t
		httpLog.Reportf("using TLS version: %s", conf.Tls.MinVersion)
	}
	srv := &Server{
		log:          httpLog,
		conf:         conf,
		httpServer:   httpServer,
		errorChannel: errorChan,
	}
	return srv
}

func (s *Server) Listen() {
	s.log.Reportf("HTTP server listening on port: %d", s.conf.Http.Port)

	go func() {
		httpErr := s.httpServer.ListenAndServe()

		if !errors.Is(httpErr, http.ErrServerClosed) {
			s.errorChannel <- fmt.Errorf("error starting HTTP server on port: %d  %s", s.conf.Http.Port, httpErr)
		}
	}()
}

func (s *Server) Shutdown() {
	s.log.Reportf("initiating server shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		s.log.Errorf("shutdown error: %v", err)
	}
	s.log.Reportf("server shutdown complete")
}
