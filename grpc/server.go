package grpc

import (
	"crypto/tls"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/grpc/proto"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/sdk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"strconv"
)

import _ "google.golang.org/grpc/encoding/gzip"

type Server struct {
	flagService  *flagService
	grpcServer   *grpc.Server
	log          log.Logger
	conf         *config.Config
	errorChannel chan error
}

func NewServer(sdkClients map[string]sdk.Client, metrics metrics.Handler, conf *config.Config, logger log.Logger, errorChan chan error) *Server {
	grpcLog := logger.WithLevel(conf.Grpc.Log.GetLevel()).WithPrefix("grpc")
	opts := make([]grpc.ServerOption, 0)
	if conf.Tls.Enabled {
		t := &tls.Config{
			MinVersion: conf.Tls.GetVersion(),
		}
		for _, c := range conf.Tls.Certificates {
			if cert, err := tls.LoadX509KeyPair(c.Cert, c.Key); err == nil {
				t.Certificates = append(t.Certificates, cert)
			}
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(t)))
		grpcLog.Reportf("using TLS version: %.1f", conf.Tls.MinVersion)
	}

	flagService := newFlagService(sdkClients, metrics, grpcLog)

	grpcServer := grpc.NewServer(opts...)
	proto.RegisterFlagServiceServer(grpcServer, flagService)

	return &Server{
		flagService:  flagService,
		log:          grpcLog,
		errorChannel: errorChan,
		grpcServer:   grpcServer,
		conf:         conf,
	}
}

func (s *Server) Listen() {
	s.log.Reportf("GRPC server listening on port: %d", s.conf.Grpc.Port)

	go func() {
		listener, err := net.Listen("tcp", ":"+strconv.Itoa(s.conf.Grpc.Port))
		if err != nil {
			s.errorChannel <- fmt.Errorf("error starting GRPC server on port: %d  %s", s.conf.Grpc.Port, err)
			return
		}
		err = s.grpcServer.Serve(listener)
		if err != nil {
			s.errorChannel <- fmt.Errorf("error starting GRPC server on port: %d  %s", s.conf.Grpc.Port, err)
			return
		}
	}()
}

func (s *Server) Shutdown() {
	s.log.Reportf("initiating server shutdown")
	s.flagService.Close()
	s.grpcServer.GracefulStop()
	s.log.Reportf("server shutdown complete")
}
