package metrics

import (
	"context"
	"errors"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"time"
)

type Handler interface {
	IncrementConnection(streamType string, streamName string)
	DecrementConnection(streamType string, streamName string)

	HttpHandler() http.Handler
}

type handler struct {
	registry        *prometheus.Registry
	responseTime    *prometheus.HistogramVec
	connectionCount *prometheus.GaugeVec
}

type Server struct {
	httpServer   *http.Server
	log          log.Logger
	conf         config.MetricsConfig
	errorChannel chan error
}

func NewServer(handler http.Handler, conf config.MetricsConfig, log log.Logger, errorChan chan error) *Server {
	metricsLog := log.WithPrefix("metrics")
	mux := http.NewServeMux()
	mux.Handle("/metrics", handler)
	metricsLog.Reportf("metrics enabled, accepting requests on path: /metrics")

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", conf.Port),
		Handler: mux,
	}

	return &Server{
		log:          metricsLog,
		httpServer:   httpServer,
		conf:         conf,
		errorChannel: errorChan,
	}
}

func NewHandler() Handler {
	reg := prometheus.NewRegistry()

	respTime := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "configcat",
		Name:      "http_request_duration_seconds",
		Help:      "Histogram of HTTP response time in seconds",
		Buckets:   prometheus.ExponentialBuckets(0.1, 1.5, 5),
	}, []string{"route", "method", "status"})

	connCount := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "configcat",
		Name:      "stream_connection_count",
		Help:      "Count of connected clients per stream",
	}, []string{"type", "stream"})

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		respTime,
		connCount,
	)

	return &handler{
		registry:        reg,
		responseTime:    respTime,
		connectionCount: connCount,
	}
}

func (h *Server) Listen() {
	h.log.Reportf("metrics HTTP server listening on port: %d", h.conf.Port)

	go func() {
		httpErr := h.httpServer.ListenAndServe()

		if !errors.Is(httpErr, http.ErrServerClosed) {
			h.errorChannel <- fmt.Errorf("error starting metrics HTTP server on port: %d  %s", h.conf.Port, httpErr)
		}
	}()
}

func (h *Server) Shutdown() {
	h.log.Reportf("initiating server shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.httpServer.Shutdown(ctx)
	if err != nil {
		h.log.Errorf("shutdown error: %v", err)
	}
	h.log.Reportf("server shutdown complete")
}

func (h *handler) HttpHandler() http.Handler {
	return promhttp.HandlerFor(h.registry, promhttp.HandlerOpts{Registry: h.registry})
}

func (h *handler) IncrementConnection(streamType string, streamName string) {
	h.connectionCount.WithLabelValues(streamType, streamName).Inc()
}

func (h *handler) DecrementConnection(streamType string, streamName string) {
	h.connectionCount.WithLabelValues(streamType, streamName).Dec()
}
