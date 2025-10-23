package telemetry

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
)

type K string
type V string

type KV struct {
	Key   K
	Value V
}

func (k K) V(val string) KV {
	return KV{
		Key:   k,
		Value: V(val),
	}
}

type HttpClientType int

type Reporter interface {
	GetPrometheusHttpHandler() http.Handler

	RecordConnections(count int64, sdkId string, streamType string, flag string)
	AddSentMessageCount(count int, sdkId string, streamType string, flag string)

	StartSpan(ctx context.Context, name string, attributes ...KV) (context.Context, trace.Span)
	ForceFlush(ctx context.Context)

	InstrumentHttp(operation string, method string, handler http.HandlerFunc) http.HandlerFunc
	InstrumentHttpClient(handler http.RoundTripper, attributes ...KV) http.RoundTripper
	InstrumentGrpc(opts []grpc.ServerOption) []grpc.ServerOption

	InstrumentRedis(rdb redis.UniversalClient)
	InstrumentMongoDb(opts *options.ClientOptions)
	InstrumentAws(opts *aws.Config)

	Shutdown()
}

const (
	traceName = "github.com/configcat/configcat-proxy"
)

type reporter struct {
	conf           *config.DiagConfig
	metricsHandler *metricsHandler
	traceHandler   *traceHandler
	tracer         trace.Tracer
	log            log.Logger
}

func NewReporter(conf *config.DiagConfig, version string, log log.Logger) Reporter {
	logger := log.WithPrefix("telemetry")
	res := buildResource(version)

	var mh *metricsHandler
	var th *traceHandler
	var tracer trace.Tracer
	if conf.IsMetricsEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		mh = newMetricsHandler(ctx, res, &conf.Metrics, logger)
	}
	if conf.IsTracesEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		th = newTraceHandler(ctx, res, &conf.Traces, logger)
		if th != nil {
			tracer = th.provider.Tracer(traceName)
		}
	}

	return &reporter{
		conf:           conf,
		metricsHandler: mh,
		traceHandler:   th,
		tracer:         tracer,
		log:            logger,
	}
}

func NewEmptyReporter() Reporter {
	return &reporter{conf: &config.DiagConfig{}}
}

func (r *reporter) GetPrometheusHttpHandler() http.Handler {
	return promhttp.Handler()
}

func (r *reporter) ForceFlush(ctx context.Context) {
	if r.metricsHandler != nil {
		err := r.metricsHandler.provider.ForceFlush(ctx)
		if err != nil {
			r.log.Errorf("failed to force flush metrics: %v", err)
		}
	}
	if r.traceHandler != nil {
		err := r.traceHandler.provider.ForceFlush(ctx)
		if err != nil {
			r.log.Errorf("failed to force flush traces: %v", err)
		}
	}
}

func (r *reporter) RecordConnections(count int64, sdkId string, streamType string, flag string) {
	if r.metricsHandler == nil {
		return
	}
	r.metricsHandler.recordConnections(count, sdkId, streamType, flag)
}

func (r *reporter) AddSentMessageCount(count int, sdkId string, streamType string, flag string) {
	if r.metricsHandler == nil {
		return
	}
	r.metricsHandler.addSentMessageCount(count, sdkId, streamType, flag)
}

func (r *reporter) StartSpan(ctx context.Context, name string, attributes ...KV) (context.Context, trace.Span) {
	if r.tracer == nil {
		return noop.NewTracerProvider().Tracer("noop").Start(ctx, "noop", trace.WithAttributes(toAttributeArray(attributes...)...))
	}
	return r.tracer.Start(ctx, name, trace.WithAttributes(toAttributeArray(attributes...)...), trace.WithSpanKind(trace.SpanKindInternal))
}

func (r *reporter) InstrumentHttp(operation string, method string, handler http.HandlerFunc) http.HandlerFunc {
	var otelOpts []otelhttp.Option
	if r.metricsHandler != nil {
		otelOpts = append(otelOpts, otelhttp.WithMeterProvider(r.metricsHandler.provider), otelhttp.WithMetricAttributesFn(func(r *http.Request) []attribute.KeyValue {
			return []attribute.KeyValue{semconv.HTTPRoute(r.URL.Path)}
		}))
	}
	if r.traceHandler != nil {
		otelOpts = append(otelOpts, otelhttp.WithTracerProvider(r.traceHandler.provider))
	}
	if len(otelOpts) > 0 {
		return otelhttp.NewHandler(handler, "HTTP "+method+" "+operation, otelOpts...).ServeHTTP
	}
	return handler
}

func (r *reporter) InstrumentHttpClient(handler http.RoundTripper, attributes ...KV) http.RoundTripper {
	var otelOpts []otelhttp.Option
	if r.metricsHandler != nil {
		otelOpts = append(otelOpts, otelhttp.WithMeterProvider(r.metricsHandler.provider))
		if len(attributes) > 0 {
			arr := toAttributeArray(attributes...)
			otelOpts = append(otelOpts, otelhttp.WithMetricAttributesFn(func(r *http.Request) []attribute.KeyValue {
				return append(arr, semconv.HTTPRoute(r.URL.Path))
			}))
		}
	}
	if r.traceHandler != nil {
		otelOpts = append(otelOpts, otelhttp.WithTracerProvider(r.traceHandler.provider))
	}
	if len(otelOpts) > 0 {
		return otelhttp.NewTransport(handler, otelOpts...)
	}
	return handler
}

func (r *reporter) InstrumentGrpc(opts []grpc.ServerOption) []grpc.ServerOption {
	var otelOpts []otelgrpc.Option
	if r.metricsHandler != nil {
		otelOpts = append(otelOpts, otelgrpc.WithMeterProvider(r.metricsHandler.provider))
	}
	if r.traceHandler != nil {
		otelOpts = append(otelOpts, otelgrpc.WithTracerProvider(r.traceHandler.provider))
	}
	if len(otelOpts) > 0 {
		return append(opts, grpc.StatsHandler(otelgrpc.NewServerHandler(otelOpts...)))
	}
	return opts
}

func (r *reporter) InstrumentRedis(rdb redis.UniversalClient) {
	if r.metricsHandler != nil {
		err := redisotel.InstrumentMetrics(rdb, redisotel.WithMeterProvider(r.metricsHandler.provider))
		if err != nil {
			r.log.Errorf("failed to instrument redis: %v", err)
		}
	}
	if r.traceHandler != nil {
		err := redisotel.InstrumentTracing(rdb, redisotel.WithTracerProvider(r.traceHandler.provider))
		if err != nil {
			r.log.Errorf("failed to instrument redis: %v", err)
		}
	}
}

func (r *reporter) InstrumentMongoDb(opts *options.ClientOptions) {
	if r.traceHandler != nil {
		opts.Monitor = otelmongo.NewMonitor(otelmongo.WithTracerProvider(r.traceHandler.provider))
	}
}

func (r *reporter) InstrumentAws(opts *aws.Config) {
	if r.traceHandler != nil {
		otelaws.AppendMiddlewares(&opts.APIOptions, otelaws.WithTracerProvider(r.traceHandler.provider))
	}
}

func (r *reporter) Shutdown() {
	if r.metricsHandler != nil {
		r.metricsHandler.shutdown()
	}
	if r.traceHandler != nil {
		r.traceHandler.Shutdown()
	}
}

func buildResource(version string) *resource.Resource {
	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("configcat-proxy"),
			semconv.ServiceVersion(version),
		))
	return res
}

func toAttributeArray(attributes ...KV) []attribute.KeyValue {
	var result []attribute.KeyValue
	for _, attr := range attributes {
		result = append(result, attribute.String(string(attr.Key), string(attr.Value)))
	}
	return result
}
