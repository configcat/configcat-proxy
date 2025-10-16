package telemetry

import (
	"testing"

	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
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
