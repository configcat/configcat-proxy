package telemetry

import (
	"context"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

type traceHandler struct {
	provider *trace.TracerProvider
	log      log.Logger
}

func newTraceHandler(ctx context.Context, resource *resource.Resource, conf *config.TraceConfig, log log.Logger) *traceHandler {
	if !conf.Otlp.Enabled {
		return nil
	}
	logger := log.WithPrefix("traces")
	providerOpts := []trace.TracerProviderOption{trace.WithResource(resource)}
	if conf.Otlp.Enabled {
		switch conf.Otlp.Protocol {
		case "grpc":
			var opts []otlptracegrpc.Option
			if conf.Otlp.Endpoint != "" {
				opts = append(opts, otlptracegrpc.WithEndpoint(conf.Otlp.Endpoint))
			}
			opts = append(opts, otlptracegrpc.WithInsecure())
			r, err := otlptracegrpc.New(ctx, opts...)
			if err != nil {
				logger.Errorf("failed to configure OTLP gRPC exporter: %s", err)
				return nil
			}
			providerOpts = append(providerOpts, trace.WithBatcher(r))
		case "http":
		case "https":
			var opts []otlptracehttp.Option
			if conf.Otlp.Endpoint != "" {
				opts = append(opts, otlptracehttp.WithEndpoint(conf.Otlp.Endpoint))
			}
			if conf.Otlp.Protocol == "http" {
				opts = append(opts, otlptracehttp.WithInsecure())
			}
			r, err := otlptracehttp.New(ctx, opts...)
			if err != nil {
				logger.Errorf("failed to configure OTLP HTTP exporter: %s", err)
				return nil
			}
			providerOpts = append(providerOpts, trace.WithBatcher(r))
		}

		var ep string
		if conf.Otlp.Endpoint != "" {
			ep = " to " + conf.Otlp.Endpoint + ""
		}
		logger.Reportf("otlp exporter enabled over %s%s", conf.Otlp.Protocol, ep)
	}
	return newTraceHandlerWithOpts(providerOpts, logger)
}

func newTraceHandlerWithOpts(opts []trace.TracerProviderOption, log log.Logger) *traceHandler {
	provider := trace.NewTracerProvider(opts...)
	return &traceHandler{
		provider: provider,
		log:      log,
	}
}

func (r *traceHandler) Shutdown() {
	r.log.Reportf("initiating server shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := r.provider.Shutdown(ctx)
	if err != nil {
		r.log.Errorf("shutdown error: %s", err)
	}
	r.log.Reportf("server shutdown complete")
}
