package telemetry

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	otlpmpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	mpb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

func TestConnection(t *testing.T) {
	reader := metric.NewManualReader()
	handler := newMetricsHandlerWithOpts([]metric.Option{metric.WithReader(reader)}, log.NewNullLogger())

	handler.recordConnections(1, "test", "t1", "n1")
	handler.recordConnections(2, "test", "t1", "n1")
	handler.recordConnections(1, "test", "t2", "n1")

	handler.addSentMessageCount(1, "test", "t1", "n1")
	handler.addSentMessageCount(4, "test", "t2", "n1")

	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)

	var sm metricdata.ScopeMetrics
	for _, s := range rm.ScopeMetrics {
		if s.Scope.Name == meterName {
			sm = s
		}
	}

	m1 := sm.Metrics[0]
	m2 := sm.Metrics[1]

	assert.Equal(t, "stream.connections", m1.Name)
	assert.Equal(t, "stream.msg.sent.total", m2.Name)

	metricdatatest.AssertEqual(t, metricdata.Metrics{
		Name:        "stream.connections",
		Description: "Number of active client connections per stream.",
		Data: metricdata.Gauge[int64]{
			DataPoints: []metricdata.DataPoint[int64]{
				{
					Value: 2,
					Attributes: attribute.NewSet(attribute.Key("sdk").String("test"),
						attribute.Key("type").String("t1"),
						attribute.Key("flag").String("n1")),
				},
				{
					Value: 1,
					Attributes: attribute.NewSet(attribute.Key("sdk").String("test"),
						attribute.Key("type").String("t2"),
						attribute.Key("flag").String("n1")),
				},
			},
		}}, m1, metricdatatest.IgnoreTimestamp())

	metricdatatest.AssertEqual(t, metricdata.Metrics{
		Name:        "stream.msg.sent.total",
		Description: "Total number of stream messages sent by the server.",
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints: []metricdata.DataPoint[int64]{
				{
					Value: 1,
					Attributes: attribute.NewSet(attribute.Key("sdk").String("test"),
						attribute.Key("type").String("t1"),
						attribute.Key("flag").String("n1")),
				},
				{
					Value: 4,
					Attributes: attribute.NewSet(attribute.Key("sdk").String("test"),
						attribute.Key("type").String("t2"),
						attribute.Key("flag").String("n1")),
				},
			},
		}}, m2, metricdatatest.IgnoreTimestamp())

	handler.recordConnections(1, "test", "t1", "n1")

	rm = metricdata.ResourceMetrics{}
	err = reader.Collect(t.Context(), &rm)
	assert.NoError(t, err)

	for _, s := range rm.ScopeMetrics {
		if s.Scope.Name == meterName {
			sm = s
		}
	}

	m1 = sm.Metrics[0]

	metricdatatest.AssertEqual(t, metricdata.Metrics{
		Name:        "stream.connections",
		Description: "Number of active client connections per stream.",
		Data: metricdata.Gauge[int64]{
			DataPoints: []metricdata.DataPoint[int64]{
				{
					Value: 1,
					Attributes: attribute.NewSet(attribute.Key("sdk").String("test"),
						attribute.Key("type").String("t1"),
						attribute.Key("flag").String("n1")),
				},
				{
					Value: 1,
					Attributes: attribute.NewSet(attribute.Key("sdk").String("test"),
						attribute.Key("type").String("t2"),
						attribute.Key("flag").String("n1")),
				},
			},
		}}, m1, metricdatatest.IgnoreTimestamp())
}

func TestOtlpMetricsExporterGrpc(t *testing.T) {
	collector, err := newInMemoryMetricGrpcCollector()
	assert.NoError(t, err)
	defer collector.Shutdown()

	handler := newMetricsHandler(t.Context(), buildResource("0.1.0"),
		&config.MetricsConfig{Enabled: true, Otlp: config.OtlpExporterConfig{Enabled: true, Protocol: "grpc", Endpoint: collector.Addr()}}, log.NewNullLogger())
	assert.NotNil(t, handler)

	handler.recordConnections(1, "test", "t1", "n1")
	_ = handler.provider.ForceFlush(t.Context())

	assert.True(t, collector.hasMetric("stream.connections"))
}

func TestOtlpMetricsExporterHttp(t *testing.T) {
	collector := newInMemoryMetricHttpCollector()
	defer collector.Shutdown()

	handler := newMetricsHandler(t.Context(), buildResource("0.1.0"),
		&config.MetricsConfig{Enabled: true, Otlp: config.OtlpExporterConfig{Enabled: true, Protocol: "http", Endpoint: collector.Addr()}}, log.NewNullLogger())
	assert.NotNil(t, handler)

	handler.recordConnections(1, "test", "t1", "n1")
	_ = handler.provider.ForceFlush(t.Context())

	assert.True(t, hasMetric(collector, "stream.connections"))
}

type inMemoryMetricGrpcCollector struct {
	otlpmpb.UnimplementedMetricsServiceServer
	*grpcCollector

	metrics []*mpb.ResourceMetrics
}

func newInMemoryMetricGrpcCollector() (*inMemoryMetricGrpcCollector, error) {
	gc, err := newGrpcCollector()
	if err != nil {
		return nil, err
	}
	c := &inMemoryMetricGrpcCollector{
		grpcCollector: gc,
		metrics:       make([]*mpb.ResourceMetrics, 0),
	}
	otlpmpb.RegisterMetricsServiceServer(c.srv, c)
	go func() { _ = c.srv.Serve(c.listener) }()

	return c, nil
}

func (c *inMemoryMetricGrpcCollector) Export(_ context.Context, req *otlpmpb.ExportMetricsServiceRequest) (*otlpmpb.ExportMetricsServiceResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = append(c.metrics, req.ResourceMetrics...)

	return &otlpmpb.ExportMetricsServiceResponse{}, nil
}

func (c *inMemoryMetricGrpcCollector) hasMetric(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, t := range c.metrics {
		for _, span := range t.ScopeMetrics {
			for _, s := range span.Metrics {
				if s.Name == name {
					return true
				}
			}
		}
	}
	return false
}

func newInMemoryMetricHttpCollector() *inMemoryHttpCollector[*otlpmpb.ExportMetricsServiceRequest] {
	c := &inMemoryHttpCollector[*otlpmpb.ExportMetricsServiceRequest]{
		records: make([]*otlpmpb.ExportMetricsServiceRequest, 0),
	}
	c.srv = httptest.NewServer(c)
	return c
}

func hasMetric(c *inMemoryHttpCollector[*otlpmpb.ExportMetricsServiceRequest], name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, r := range c.records {
		for _, t := range r.ResourceMetrics {
			for _, span := range t.ScopeMetrics {
				for _, s := range span.Metrics {
					if s.Name == name {
						return true
					}
				}
			}
		}
	}
	return false
}
