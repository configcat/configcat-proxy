package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Reporter interface {
	IncrementConnection(sdkId string, streamType string, flag string)
	DecrementConnection(sdkId string, streamType string, flag string)
	AddSentMessageCount(count int, sdkId string, streamType string, flag string)

	HttpHandler() http.Handler
}

type reporter struct {
	registry            *prometheus.Registry
	httpResponseTime    *prometheus.HistogramVec
	grpcResponseTime    *prometheus.HistogramVec
	sdkResponseTime     *prometheus.HistogramVec
	profileResponseTime *prometheus.HistogramVec
	connections         *prometheus.GaugeVec
	streamMessageSent   *prometheus.CounterVec
}

func NewReporter() Reporter {
	reg := prometheus.NewRegistry()

	httpRespTime := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "configcat",
		Name:      "http_request_duration_seconds",
		Help:      "Histogram of Proxy HTTP response time in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"route", "method", "status"})

	grpcRespTime := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "configcat",
		Name:      "grpc_rpc_duration_seconds",
		Help:      "Histogram of RPC response latency in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "code"})

	sdkRespTime := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "configcat",
		Name:      "sdk_http_request_duration_seconds",
		Help:      "Histogram of ConfigCat CDN HTTP response time in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"sdk", "route", "status"})

	profileResponseTime := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "configcat",
		Name:      "profile_http_request_duration_seconds",
		Help:      "Histogram of Proxy profile HTTP response time in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"key", "route", "status"})

	connections := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "configcat",
		Name:      "stream_connections",
		Help:      "Number of active client connections per stream.",
	}, []string{"sdk", "type", "flag"})

	streamMessageSent := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "configcat",
		Name:      "stream_msg_sent_total",
		Help:      "Total number of stream messages sent by the server.",
	}, []string{"sdk", "type", "flag"})

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		httpRespTime,
		grpcRespTime,
		sdkRespTime,
		profileResponseTime,
		connections,
		streamMessageSent,
	)

	return &reporter{
		registry:            reg,
		httpResponseTime:    httpRespTime,
		grpcResponseTime:    grpcRespTime,
		sdkResponseTime:     sdkRespTime,
		profileResponseTime: profileResponseTime,
		connections:         connections,
		streamMessageSent:   streamMessageSent,
	}
}

func (r *reporter) HttpHandler() http.Handler {
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{Registry: r.registry})
}

func (r *reporter) IncrementConnection(sdkId string, streamType string, flag string) {
	r.connections.WithLabelValues(sdkId, streamType, flag).Inc()
}

func (r *reporter) DecrementConnection(sdkId string, streamType string, flag string) {
	r.connections.WithLabelValues(sdkId, streamType, flag).Dec()
}

func (r *reporter) AddSentMessageCount(count int, sdkId string, streamType string, flag string) {
	r.streamMessageSent.WithLabelValues(sdkId, streamType, flag).Add(float64(count))
}
