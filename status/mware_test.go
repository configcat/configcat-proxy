package status

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInterceptSdk(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		reporter := NewReporter(config.Config{}).(*reporter)
		repSrv := httptest.NewServer(reporter.HttpHandler())
		h := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(h)
		client := http.Client{}
		client.Transport = InterceptSdk(reporter, http.DefaultTransport)
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		_, _ = client.Do(req)

		stat := readStatus(repSrv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDK.Source.Status)
		assert.Equal(t, Online, stat.SDK.Mode)
		assert.Equal(t, 1, len(stat.SDK.Source.Records))
		assert.Contains(t, stat.SDK.Source.Records[0], "config fetched")
		assert.Equal(t, RemoteSrc, stat.SDK.Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("not modified", func(t *testing.T) {
		reporter := NewReporter(config.Config{}).(*reporter)
		repSrv := httptest.NewServer(reporter.HttpHandler())
		h := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusNotModified)
		})
		srv := httptest.NewServer(h)
		client := http.Client{}
		client.Transport = InterceptSdk(reporter, http.DefaultTransport)
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		_, _ = client.Do(req)

		stat := readStatus(repSrv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDK.Source.Status)
		assert.Equal(t, Online, stat.SDK.Mode)
		assert.Equal(t, 1, len(stat.SDK.Source.Records))
		assert.Contains(t, stat.SDK.Source.Records[0], "config not modified")
		assert.Equal(t, RemoteSrc, stat.SDK.Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("error", func(t *testing.T) {
		reporter := NewReporter(config.Config{}).(*reporter)
		repSrv := httptest.NewServer(reporter.HttpHandler())
		h := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusBadRequest)
		})
		srv := httptest.NewServer(h)
		client := http.Client{}
		client.Transport = InterceptSdk(reporter, http.DefaultTransport)
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		_, _ = client.Do(req)

		stat := readStatus(repSrv.URL)

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Degraded, stat.SDK.Source.Status)
		assert.Equal(t, Online, stat.SDK.Mode)
		assert.Equal(t, 1, len(stat.SDK.Source.Records))
		assert.Contains(t, stat.SDK.Source.Records[0], "unexpected response received: 400 Bad Request")
		assert.Equal(t, RemoteSrc, stat.SDK.Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
}
