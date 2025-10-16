package telemetry

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestHandler_With_Prometheus(t *testing.T) {
	conf := config.DiagConfig{
		Port:    5052,
		Enabled: true,
		Status:  config.StatusConfig{Enabled: false},
		Metrics: config.MetricsConfig{Enabled: true, Prometheus: config.PrometheusExporterConfig{Enabled: true}},
	}

	handler := NewReporter(&conf, "0.1.0", log.NewNullLogger()).(*reporter)

	mSrv := httptest.NewServer(promhttp.Handler())

	t.Run("http", func(t *testing.T) {
		h := handler.InstrumentHttp("t", http.MethodGet, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(h)
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		_, _ = http.DefaultClient.Do(req)

		req, _ = http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		assert.Contains(t, string(body), "configcat_http_server_request_duration_seconds_bucket{http_request_method=\"GET\",http_response_status_code=\"200\",http_route=\"/\",network_protocol_name=\"http\",network_protocol_version=\"1.1\"")
	})
	t.Run("http bad", func(t *testing.T) {
		h := handler.InstrumentHttp("t", http.MethodGet, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusBadRequest)
		})
		srv := httptest.NewServer(h)
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		_, _ = http.DefaultClient.Do(req)

		req, _ = http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		assert.Contains(t, string(body), "configcat_http_server_request_duration_seconds_bucket{http_request_method=\"GET\",http_response_status_code=\"400\",http_route=\"/\",network_protocol_name=\"http\",network_protocol_version=\"1.1\"")
	})
	t.Run("http client", func(t *testing.T) {
		h := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(h)
		client := http.Client{}
		client.Transport = handler.InstrumentHttpClient(http.DefaultTransport, NewKV("sdk", "test"))
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		_, _ = client.Do(req)

		req, _ = http.NewRequest(http.MethodGet, mSrv.URL, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		assert.Contains(t, string(body), "configcat_http_client_request_duration_seconds_bucket{http_request_method=\"GET\",http_response_status_code=\"200\",http_route=\"\",network_protocol_name=\"http\"")
	})
	t.Run("grpc", func(t *testing.T) {
		opts := handler.InstrumentGrpc([]grpc.ServerOption{})
		assert.Equal(t, 1, len(opts))
	})
}
