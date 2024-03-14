package grpc

import (
	"crypto/tls"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/metrics"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/grpc/proto"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"net"
	"strconv"
	"time"
)

import _ "google.golang.org/grpc/encoding/gzip"

type Server struct {
	flagService       *flagService
	grpcServer        *grpc.Server
	healthServer      *health.Server
	log               log.Logger
	conf              *config.Config
	statusReporter    status.Reporter
	healthCheckTicker *time.Ticker
	stop              chan struct{}
	errorChannel      chan error
}

func NewServer(sdkClients map[string]sdk.Client, metricsReporter metrics.Reporter, statusReporter status.Reporter, conf *config.Config, logger log.Logger, errorChan chan error) (*Server, error) {
	grpcLog := logger.WithLevel(conf.Grpc.Log.GetLevel()).WithPrefix("grpc")
	opts := make([]grpc.ServerOption, 0)
	if conf.Tls.Enabled {
		t := &tls.Config{
			MinVersion: conf.Tls.GetVersion(),
		}
		for _, c := range conf.Tls.Certificates {
			if cert, err := tls.LoadX509KeyPair(c.Cert, c.Key); err == nil {
				t.Certificates = append(t.Certificates, cert)
			} else {
				grpcLog.Errorf("failed to load the certificate and key pair: %s", err)
				return nil, err
			}
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(t)))
		grpcLog.Reportf("using TLS version: %.1f", conf.Tls.MinVersion)
	}

	unaryInterceptors := make([]grpc.UnaryServerInterceptor, 0)
	streamInterceptors := make([]grpc.StreamServerInterceptor, 0)
	if metricsReporter != nil {
		unaryInterceptors = append(unaryInterceptors, metrics.GrpcUnaryInterceptor(metricsReporter))
	}
	if grpcLog.Level() == log.Debug {
		unaryInterceptors = append(unaryInterceptors, DebugLogUnaryInterceptor(grpcLog))
		streamInterceptors = append(streamInterceptors, DebugLogStreamInterceptor(grpcLog))
	}
	if len(unaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(unaryInterceptors...))
	}
	if len(streamInterceptors) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(streamInterceptors...))
	}
	if params, ok := conf.Grpc.KeepAlive.ToParams(); ok {
		opts = append(opts, grpc.KeepaliveParams(params))
	}

	flagService := newFlagService(sdkClients, metricsReporter, grpcLog)

	grpcServer := grpc.NewServer(opts...)
	proto.RegisterFlagServiceServer(grpcServer, flagService)
	var healthServer *health.Server
	if conf.Grpc.HealthCheckEnabled {
		healthServer = health.NewServer()
		grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	}
	if conf.Grpc.ServerReflectionEnabled {
		reflection.Register(grpcServer)
	}

	srv := &Server{
		flagService:    flagService,
		log:            grpcLog,
		errorChannel:   errorChan,
		grpcServer:     grpcServer,
		healthServer:   healthServer,
		statusReporter: statusReporter,
		conf:           conf,
		stop:           make(chan struct{}),
	}
	if healthServer != nil {
		srv.healthCheckTicker = time.NewTicker(5 * time.Second)
	}
	return srv, nil
}

func (s *Server) Listen() {
	s.log.Reportf("GRPC server listening on port: %d", s.conf.Grpc.Port)

	go s.runHealthCheck()
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

func (s *Server) runHealthCheck() {
	if s.healthCheckTicker == nil || s.healthServer == nil {
		return
	}

	select {
	case <-s.healthCheckTicker.C:
		stat := s.statusReporter.GetStatus().Status
		hcResp := grpc_health_v1.HealthCheckResponse_SERVING
		if stat == status.Down {
			hcResp = grpc_health_v1.HealthCheckResponse_NOT_SERVING
		}
		s.healthServer.SetServingStatus("", hcResp)
	case <-s.stop:
		return
	}
}

func (s *Server) Shutdown() {
	s.log.Reportf("initiating server shutdown")
	close(s.stop)
	if s.healthCheckTicker != nil {
		s.healthCheckTicker.Stop()
	}
	s.flagService.Close()
	s.grpcServer.GracefulStop()
	s.log.Reportf("server shutdown complete")
}
