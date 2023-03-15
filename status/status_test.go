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
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {}}})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportOk("t", "")
		stat := readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.Environments["t"].Source.Status)
		assert.Equal(t, Online, stat.Environments["t"].Mode)
		assert.Equal(t, 1, len(stat.Environments["t"].Source.Records))
		assert.Equal(t, RemoteSrc, stat.Environments["t"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})

	t.Run("degraded after 2 errors, then ok again", func(t *testing.T) {
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {}}})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportError("t", fmt.Errorf(""))
		reporter.ReportError("t", fmt.Errorf(""))
		stat := readStatus(srv.URL)

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Degraded, stat.Environments["t"].Source.Status)
		assert.Equal(t, Online, stat.Environments["t"].Mode)
		assert.Equal(t, 2, len(stat.Environments["t"].Source.Records))
		assert.Equal(t, RemoteSrc, stat.Environments["t"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))

		reporter.ReportOk("t", "")
		stat = readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.Environments["t"].Source.Status)
		assert.Equal(t, Online, stat.Environments["t"].Mode)
		assert.Equal(t, 3, len(stat.Environments["t"].Source.Records))
		assert.Equal(t, RemoteSrc, stat.Environments["t"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("max 5 records", func(t *testing.T) {
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {}}})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportOk("t", "m1")
		reporter.ReportOk("t", "m2")
		reporter.ReportOk("t", "m3")
		reporter.ReportOk("t", "m4")
		reporter.ReportOk("t", "m5")
		reporter.ReportOk("t", "m6")
		stat := readStatus(srv.URL)

		assert.Equal(t, 5, len(stat.Environments["t"].Source.Records))
		assert.Contains(t, stat.Environments["t"].Source.Records[0], "m2")
		assert.Contains(t, stat.Environments["t"].Source.Records[1], "m3")
		assert.Contains(t, stat.Environments["t"].Source.Records[2], "m4")
		assert.Contains(t, stat.Environments["t"].Source.Records[3], "m5")
		assert.Contains(t, stat.Environments["t"].Source.Records[4], "m6")
	})
}

func TestReporter_Offline(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: "test"}}}}})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportOk("t", "")
		stat := readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.Environments["t"].Source.Status)
		assert.Equal(t, Offline, stat.Environments["t"].Mode)
		assert.Equal(t, 1, len(stat.Environments["t"].Source.Records))
		assert.Equal(t, FileSrc, stat.Environments["t"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("cache invalid", func(t *testing.T) {
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {Offline: config.OfflineConfig{Enabled: true, UseCache: true}}}})
		srv := httptest.NewServer(reporter.HttpHandler())
		stat := readStatus(srv.URL)

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Degraded, stat.Environments["t"].Source.Status)
		assert.Equal(t, Offline, stat.Environments["t"].Mode)
		assert.Equal(t, 1, len(stat.Environments["t"].Source.Records))
		assert.Equal(t, CacheSrc, stat.Environments["t"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("cache valid", func(t *testing.T) {
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {Offline: config.OfflineConfig{Enabled: true, UseCache: true}}}, Cache: config.CacheConfig{Redis: config.RedisConfig{Enabled: true}}})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportOk("t", "")
		reporter.ReportOk(Cache, "")
		stat := readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.Environments["t"].Source.Status)
		assert.Equal(t, Offline, stat.Environments["t"].Mode)
		assert.Equal(t, 1, len(stat.Environments["t"].Source.Records))
		assert.Equal(t, CacheSrc, stat.Environments["t"].Source.Type)
		assert.Equal(t, Healthy, stat.Cache.Status)
		assert.Equal(t, 1, len(stat.Cache.Records))
	})
}

func TestReporter_StatusCopy(t *testing.T) {
	reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {}}}).(*reporter)
	reporter.ReportError("t", fmt.Errorf(""))
	stat := reporter.getStatus()

	assert.Equal(t, Healthy, reporter.status.Status)
	assert.Equal(t, Degraded, stat.Status)
}

func TestReporter_Degraded_Calc(t *testing.T) {
	t.Run("1 record, 1 error", func(t *testing.T) {
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {}}}).(*reporter)
		reporter.ReportError("t", fmt.Errorf(""))
		stat := reporter.getStatus()

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Degraded, stat.Environments["t"].Source.Status)
	})
	t.Run("2 records, 1 error then 1 ok", func(t *testing.T) {
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {}}}).(*reporter)
		reporter.ReportError("t", fmt.Errorf(""))
		reporter.ReportOk("t", "")
		stat := reporter.getStatus()

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.Environments["t"].Source.Status)
	})
	t.Run("2 records, 1 ok then 1 error", func(t *testing.T) {
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {}}}).(*reporter)
		reporter.ReportOk("t", "")
		reporter.ReportError("t", fmt.Errorf(""))
		stat := reporter.getStatus()

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.Environments["t"].Source.Status)
	})
	t.Run("3 records, 1 ok then 2 errors", func(t *testing.T) {
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {}}}).(*reporter)
		reporter.ReportOk("t", "")
		reporter.ReportError("t", fmt.Errorf(""))
		reporter.ReportError("t", fmt.Errorf(""))
		stat := reporter.getStatus()

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Degraded, stat.Environments["t"].Source.Status)
	})
	t.Run("3 records, 1 ok then 1 error then 1 ok", func(t *testing.T) {
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {}}}).(*reporter)
		reporter.ReportOk("t", "")
		reporter.ReportError("t", fmt.Errorf(""))
		reporter.ReportOk("t", "")
		stat := reporter.getStatus()

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.Environments["t"].Source.Status)
	})
	t.Run("3 records, 1 error then 1 ok then 1 error", func(t *testing.T) {
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t": {}}}).(*reporter)
		reporter.ReportError("t", fmt.Errorf(""))
		reporter.ReportOk("t", "")
		reporter.ReportError("t", fmt.Errorf(""))
		stat := reporter.getStatus()

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.Environments["t"].Source.Status)
	})
	t.Run("2 envs 1 degraded", func(t *testing.T) {
		reporter := NewReporter(&config.Config{Environments: map[string]*config.SDKConfig{"t1": {}, "t2": {}}}).(*reporter)
		reporter.ReportError("t1", fmt.Errorf(""))
		reporter.ReportOk("t2", "")
		stat := reporter.getStatus()

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Degraded, stat.Environments["t1"].Source.Status)
		assert.Equal(t, Healthy, stat.Environments["t2"].Source.Status)
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
