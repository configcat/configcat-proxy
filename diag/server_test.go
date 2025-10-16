package diag

import (
	"net/http"
	"testing"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	errChan := make(chan error)
	conf := config.DiagConfig{
		Port:    5051,
		Enabled: true,
		Status:  config.StatusConfig{Enabled: true},
		Metrics: config.MetricsConfig{Enabled: true, Prometheus: config.PrometheusExporterConfig{Enabled: true}},
	}

	reporter := status.NewEmptyReporter()
	reporter.RegisterSdk("test", &config.SDKConfig{Key: "key"})
	srv := NewServer(&conf, telemetry.NewReporter(&conf, "0.1.0", log.NewNullLogger()), reporter, log.NewNullLogger(), errChan)
	srv.Listen()
	time.Sleep(500 * time.Millisecond)

	req, _ := http.NewRequest(http.MethodGet, "http://localhost:5051/status", http.NoBody)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	req, _ = http.NewRequest(http.MethodGet, "http://localhost:5051/metrics", http.NoBody)
	resp, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	srv.Shutdown()

	assert.Nil(t, readFromErrChan(errChan))
}

func TestNewServer_NotEnabled(t *testing.T) {
	errChan := make(chan error)
	conf := config.DiagConfig{
		Port:    5052,
		Enabled: true,
		Status:  config.StatusConfig{Enabled: false},
		Metrics: config.MetricsConfig{Enabled: false, Prometheus: config.PrometheusExporterConfig{Enabled: true}},
	}

	reporter := status.NewEmptyReporter()
	reporter.RegisterSdk("test", &config.SDKConfig{Key: "key"})
	srv := NewServer(&conf, telemetry.NewReporter(&conf, "0.1.0", log.NewNullLogger()), reporter, log.NewNullLogger(), errChan)
	srv.Listen()
	time.Sleep(500 * time.Millisecond)

	req, _ := http.NewRequest(http.MethodGet, "http://localhost:5052/status", http.NoBody)
	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	req, _ = http.NewRequest(http.MethodGet, "http://localhost:5052/metrics", http.NoBody)
	resp, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	srv.Shutdown()

	assert.Nil(t, readFromErrChan(errChan))
}

func readFromErrChan(ch chan error) error {
	select {
	case val, ok := <-ch:
		if ok {
			return val
		}
	default:
		return nil
	}
	return nil
}
