package telemetry

import (
	"context"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/prometheus/otlptranslator"
	otelhost "go.opentelemetry.io/contrib/instrumentation/host"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

type metricsHandler struct {
	connections       otelmetric.Int64Gauge
	streamMessageSent otelmetric.Int64Counter
	provider          *metric.MeterProvider
	log               log.Logger

	ctx       context.Context
	ctxCancel func()
}

const (
	meterName = "github.com/configcat/configcat-proxy"
)

func newMetricsHandler(ctx context.Context, resource *resource.Resource, conf *config.MetricsConfig, log log.Logger) *metricsHandler {
	if !conf.Prometheus.Enabled && !conf.Otlp.Enabled {
		return nil
	}
	logger := log.WithPrefix("metrics")
	providerOpts := []metric.Option{metric.WithResource(resource)}
	if conf.Prometheus.Enabled {
		exporter, err := promexporter.New(
			promexporter.WithNamespace("configcat"),
			promexporter.WithTranslationStrategy(otlptranslator.UnderscoreEscapingWithSuffixes))
		if err != nil {
			logger.Errorf("failed to configure Prometheus exporter: %s", err)
			return nil
		}
		providerOpts = append(providerOpts, metric.WithReader(exporter))
		logger.Reportf("prometheus exporter enabled on /metrics")
	}
	if conf.Otlp.Enabled {
		switch conf.Otlp.Protocol {
		case "grpc":
			var opts []otlpmetricgrpc.Option
			if conf.Otlp.Endpoint != "" {
				opts = append(opts, otlpmetricgrpc.WithEndpoint(conf.Otlp.Endpoint))
			}
			opts = append(opts, otlpmetricgrpc.WithInsecure())
			r, err := otlpmetricgrpc.New(ctx, opts...)
			if err != nil {
				logger.Errorf("failed to configure OTLP gRPC exporter: %s", err)
				return nil
			}
			providerOpts = append(providerOpts, metric.WithReader(metric.NewPeriodicReader(r)))
		case "http":
			fallthrough
		case "https":
			var opts []otlpmetrichttp.Option
			if conf.Otlp.Endpoint != "" {
				opts = append(opts, otlpmetrichttp.WithEndpoint(conf.Otlp.Endpoint))
			}
			if conf.Otlp.Protocol == "http" {
				opts = append(opts, otlpmetrichttp.WithInsecure())
			}
			r, err := otlpmetrichttp.New(ctx, opts...)
			if err != nil {
				logger.Errorf("failed to configure OTLP HTTP exporter: %s", err)
				return nil
			}
			providerOpts = append(providerOpts, metric.WithReader(metric.NewPeriodicReader(r)))
		}
		var ep string
		if conf.Otlp.Endpoint != "" {
			ep = " to " + conf.Otlp.Endpoint + ""
		}
		logger.Reportf("otlp exporter enabled over %s%s", conf.Otlp.Protocol, ep)
	}
	return newMetricsHandlerWithOpts(providerOpts, logger)
}

func newMetricsHandlerWithOpts(opts []metric.Option, logger log.Logger) *metricsHandler {
	provider := metric.NewMeterProvider(opts...)
	meter := provider.Meter(meterName)

	err := otelruntime.Start(otelruntime.WithMeterProvider(provider))
	if err != nil {
		logger.Errorf("failed to start runtime metrics: %s", err)
	}
	err = otelhost.Start(otelhost.WithMeterProvider(provider))
	if err != nil {
		logger.Errorf("failed to start host metrics: %s", err)
	}

	connections, err := meter.Int64Gauge("stream.connections",
		otelmetric.WithDescription("Number of active client connections per stream."))
	if err != nil {
		logger.Errorf("failed to configure connections gauge: %s", err)
		return nil
	}

	streamMessageSent, err := meter.Int64Counter("stream.msg.sent.total",
		otelmetric.WithDescription("Total number of stream messages sent by the server."))
	if err != nil {
		logger.Errorf("failed to configure stream message sent counter: %s", err)
		return nil
	}

	ctx, ctxCancel := context.WithCancel(context.Background())

	return &metricsHandler{
		connections:       connections,
		streamMessageSent: streamMessageSent,
		provider:          provider,
		log:               logger,
		ctx:               ctx,
		ctxCancel:         ctxCancel,
	}
}

func (r *metricsHandler) recordConnections(count int64, sdkId string, streamType string, flag string) {
	r.connections.Record(r.ctx, count, otelmetric.WithAttributes(
		attribute.Key("sdk").String(sdkId),
		attribute.Key("type").String(streamType),
		attribute.Key("flag").String(flag),
	))
}

func (r *metricsHandler) addSentMessageCount(count int, sdkId string, streamType string, flag string) {
	r.streamMessageSent.Add(r.ctx, int64(count), otelmetric.WithAttributes(
		attribute.Key("sdk").String(sdkId),
		attribute.Key("type").String(streamType),
		attribute.Key("flag").String(flag),
	))
}

func (r *metricsHandler) shutdown() {
	r.log.Reportf("initiating server shutdown")
	r.ctxCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := r.provider.Shutdown(ctx)
	if err != nil {
		r.log.Errorf("shutdown error: %s", err)
	}
	r.log.Reportf("server shutdown complete")
}
