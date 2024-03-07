package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type Reporter interface {
	IncrementConnection(sdkId string, streamType string, flag string)
	DecrementConnection(sdkId string, streamType string, flag string)

	HttpHandler() http.Handler
}

type reporter struct {
	registry        *prometheus.Registry
	responseTime    *prometheus.HistogramVec
	sdkResponseTime *prometheus.HistogramVec
	connections     *prometheus.GaugeVec
}

func NewReporter() Reporter {
	reg := prometheus.NewRegistry()

	respTime := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "configcat",
		Name:      "http_request_duration_seconds",
		Help:      "Histogram of Proxy HTTP response time in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"route", "method", "status"})

	sdkRespTime := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "configcat",
		Name:      "sdk_http_request_duration_seconds",
		Help:      "Histogram of ConfigCat CDN HTTP response time in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"sdk", "route", "status"})

	connections := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "configcat",
		Name:      "stream_connections",
		Help:      "Number of active client connections per stream.",
	}, []string{"sdk", "type", "flag"})

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		respTime,
		sdkRespTime,
		connections,
	)

	return &reporter{
		registry:        reg,
		responseTime:    respTime,
		sdkResponseTime: sdkRespTime,
		connections:     connections,
	}
}

func (h *reporter) HttpHandler() http.Handler {
	return promhttp.HandlerFor(h.registry, promhttp.HandlerOpts{Registry: h.registry})
}

func (h *reporter) IncrementConnection(sdkId string, streamType string, flag string) {
	h.connections.WithLabelValues(sdkId, streamType, flag).Inc()
}

func (h *reporter) DecrementConnection(sdkId string, streamType string, flag string) {
	h.connections.WithLabelValues(sdkId, streamType, flag).Dec()
}
