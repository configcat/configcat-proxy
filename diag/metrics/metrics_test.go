package metrics

import (
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConnection(t *testing.T) {
	handler := NewReporter().(*reporter)

	handler.IncrementConnection("test", "t1", "n1")
	handler.IncrementConnection("test", "t1", "n1")
	handler.IncrementConnection("test", "t2", "n1")
	handler.IncrementConnection("test", "t2", "n1")
	handler.IncrementConnection("test", "t2", "n1")

	handler.AddSentMessageCount(1, "test", "t1", "n1")
	handler.AddSentMessageCount(4, "test", "t2", "n1")

	assert.Equal(t, 2, testutil.CollectAndCount(handler.connections))
	assert.Equal(t, 2, testutil.CollectAndCount(handler.streamMessageSent))

	assert.Equal(t, float64(2), testutil.ToFloat64(handler.connections.WithLabelValues("test", "t1", "n1")))
	assert.Equal(t, float64(3), testutil.ToFloat64(handler.connections.WithLabelValues("test", "t2", "n1")))

	assert.Equal(t, float64(1), testutil.ToFloat64(handler.streamMessageSent.WithLabelValues("test", "t1", "n1")))
	assert.Equal(t, float64(4), testutil.ToFloat64(handler.streamMessageSent.WithLabelValues("test", "t2", "n1")))

	handler.DecrementConnection("test", "t1", "n1")
	handler.DecrementConnection("test", "t2", "n1")
	handler.DecrementConnection("test", "t2", "n1")

	assert.Equal(t, float64(1), testutil.ToFloat64(handler.connections.WithLabelValues("test", "t1", "n1")))
	assert.Equal(t, float64(1), testutil.ToFloat64(handler.connections.WithLabelValues("test", "t2", "n1")))
}
