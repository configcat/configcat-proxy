package status

import (
	"encoding/json"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReporter_Online(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		reporter := NewReporter(config.Config{})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportOk(SDK, "")
		stat := readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDK.Source.Status)
		assert.Equal(t, Online, stat.SDK.Mode)
		assert.Equal(t, 1, len(stat.SDK.Source.Records))
		assert.Equal(t, RemoteSrc, stat.SDK.Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})

	t.Run("degraded after 2 errors, then ok again", func(t *testing.T) {
		reporter := NewReporter(config.Config{})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportError(SDK, fmt.Errorf(""))
		reporter.ReportError(SDK, fmt.Errorf(""))
		stat := readStatus(srv.URL)

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Degraded, stat.SDK.Source.Status)
		assert.Equal(t, Online, stat.SDK.Mode)
		assert.Equal(t, 2, len(stat.SDK.Source.Records))
		assert.Equal(t, RemoteSrc, stat.SDK.Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))

		reporter.ReportOk(SDK, "")
		stat = readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDK.Source.Status)
		assert.Equal(t, Online, stat.SDK.Mode)
		assert.Equal(t, 3, len(stat.SDK.Source.Records))
		assert.Equal(t, RemoteSrc, stat.SDK.Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("max 5 records", func(t *testing.T) {
		reporter := NewReporter(config.Config{})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportOk(SDK, "m1")
		reporter.ReportOk(SDK, "m2")
		reporter.ReportOk(SDK, "m3")
		reporter.ReportOk(SDK, "m4")
		reporter.ReportOk(SDK, "m5")
		reporter.ReportOk(SDK, "m6")
		stat := readStatus(srv.URL)

		assert.Equal(t, 5, len(stat.SDK.Source.Records))
		assert.Contains(t, stat.SDK.Source.Records[0], "m2")
		assert.Contains(t, stat.SDK.Source.Records[1], "m3")
		assert.Contains(t, stat.SDK.Source.Records[2], "m4")
		assert.Contains(t, stat.SDK.Source.Records[3], "m5")
		assert.Contains(t, stat.SDK.Source.Records[4], "m6")
	})
}

func TestReporter_Offline(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		reporter := NewReporter(config.Config{SDK: config.SDKConfig{Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: "test"}}}})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportOk(SDK, "")
		stat := readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDK.Source.Status)
		assert.Equal(t, Offline, stat.SDK.Mode)
		assert.Equal(t, 1, len(stat.SDK.Source.Records))
		assert.Equal(t, FileSrc, stat.SDK.Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("cache invalid", func(t *testing.T) {
		reporter := NewReporter(config.Config{SDK: config.SDKConfig{Offline: config.OfflineConfig{Enabled: true, UseCache: true}}})
		srv := httptest.NewServer(reporter.HttpHandler())
		stat := readStatus(srv.URL)

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Degraded, stat.SDK.Source.Status)
		assert.Equal(t, Offline, stat.SDK.Mode)
		assert.Equal(t, 1, len(stat.SDK.Source.Records))
		assert.Equal(t, CacheSrc, stat.SDK.Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("cache valid", func(t *testing.T) {
		reporter := NewReporter(config.Config{SDK: config.SDKConfig{Offline: config.OfflineConfig{Enabled: true, UseCache: true}, Cache: config.CacheConfig{Redis: config.RedisConfig{Enabled: true}}}})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportOk(SDK, "")
		reporter.ReportOk(Cache, "")
		stat := readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDK.Source.Status)
		assert.Equal(t, Offline, stat.SDK.Mode)
		assert.Equal(t, 1, len(stat.SDK.Source.Records))
		assert.Equal(t, CacheSrc, stat.SDK.Source.Type)
		assert.Equal(t, Healthy, stat.Cache.Status)
		assert.Equal(t, 1, len(stat.Cache.Records))
	})
}

func readStatus(url string) Status {
	client := http.Client{}
	req, _ := http.NewRequest(http.MethodGet, url, http.NoBody)
	resp, _ := client.Do(req)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	var stat Status
	_ = json.Unmarshal(body, &stat)
	return stat
}
