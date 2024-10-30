package status

import (
	"encoding/json"
	"github.com/configcat/configcat-proxy/config"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReporter_Online(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		reporter := NewEmptyReporter()
		reporter.RegisterSdk("t", &config.SDKConfig{})
		srv := httptest.NewServer(reporter.HttpHandler())
		stat := readStatus(srv.URL)

		assert.Equal(t, Initializing, stat.Status)
		assert.Equal(t, Initializing, stat.SDKs["t"].Source.Status)
		assert.Equal(t, Online, stat.SDKs["t"].Mode)
		assert.Equal(t, 0, len(stat.SDKs["t"].Source.Records))
		assert.Equal(t, RemoteSrc, stat.SDKs["t"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))

		reporter.ReportOk("t", "")
		stat = readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDKs["t"].Source.Status)
		assert.Equal(t, Online, stat.SDKs["t"].Mode)
		assert.Equal(t, 1, len(stat.SDKs["t"].Source.Records))
		assert.Equal(t, RemoteSrc, stat.SDKs["t"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("down after 1 error, then ok, then degraded", func(t *testing.T) {
		reporter := NewEmptyReporter()
		reporter.RegisterSdk("t", &config.SDKConfig{})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportError("t", "")
		stat := readStatus(srv.URL)

		assert.Equal(t, Down, stat.Status)
		assert.Equal(t, Down, stat.SDKs["t"].Source.Status)
		assert.Equal(t, Online, stat.SDKs["t"].Mode)
		assert.Equal(t, 1, len(stat.SDKs["t"].Source.Records))
		assert.Equal(t, RemoteSrc, stat.SDKs["t"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))

		reporter.ReportOk("t", "")
		stat = readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDKs["t"].Source.Status)
		assert.Equal(t, Online, stat.SDKs["t"].Mode)
		assert.Equal(t, 2, len(stat.SDKs["t"].Source.Records))
		assert.Equal(t, RemoteSrc, stat.SDKs["t"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))

		reporter.ReportError("t", "")
		reporter.ReportError("t", "")
		stat = readStatus(srv.URL)

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Degraded, stat.SDKs["t"].Source.Status)
		assert.Equal(t, Online, stat.SDKs["t"].Mode)
		assert.Equal(t, 4, len(stat.SDKs["t"].Source.Records))
		assert.Equal(t, RemoteSrc, stat.SDKs["t"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("2 sdks (1 ok, 1 down) degraded, then ok after remove faulty sdk", func(t *testing.T) {
		reporter := NewEmptyReporter()
		reporter.RegisterSdk("t1", &config.SDKConfig{})
		reporter.RegisterSdk("t2", &config.SDKConfig{})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportOk("t1", "")
		reporter.ReportError("t2", "")
		stat := readStatus(srv.URL)

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Healthy, stat.SDKs["t1"].Source.Status)
		assert.Equal(t, Down, stat.SDKs["t2"].Source.Status)
		assert.Equal(t, Online, stat.SDKs["t1"].Mode)
		assert.Equal(t, Online, stat.SDKs["t2"].Mode)
		assert.Equal(t, 1, len(stat.SDKs["t1"].Source.Records))
		assert.Equal(t, 1, len(stat.SDKs["t2"].Source.Records))
		assert.Equal(t, RemoteSrc, stat.SDKs["t1"].Source.Type)
		assert.Equal(t, RemoteSrc, stat.SDKs["t2"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))

		reporter.RemoveSdk("t2")
		stat = readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, 1, len(stat.SDKs))
		assert.Equal(t, Healthy, stat.SDKs["t1"].Source.Status)
		assert.Equal(t, Online, stat.SDKs["t1"].Mode)
		assert.Equal(t, 1, len(stat.SDKs["t1"].Source.Records))
		assert.Equal(t, RemoteSrc, stat.SDKs["t1"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("max 5 records", func(t *testing.T) {
		reporter := NewEmptyReporter()
		reporter.RegisterSdk("t", &config.SDKConfig{})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportOk("t", "m1")
		reporter.ReportOk("t", "m2")
		reporter.ReportOk("t", "m3")
		reporter.ReportOk("t", "m4")
		reporter.ReportOk("t", "m5")
		reporter.ReportOk("t", "m6")
		stat := readStatus(srv.URL)

		assert.Equal(t, 5, len(stat.SDKs["t"].Source.Records))
		assert.Contains(t, stat.SDKs["t"].Source.Records[0], "m2")
		assert.Contains(t, stat.SDKs["t"].Source.Records[1], "m3")
		assert.Contains(t, stat.SDKs["t"].Source.Records[2], "m4")
		assert.Contains(t, stat.SDKs["t"].Source.Records[3], "m5")
		assert.Contains(t, stat.SDKs["t"].Source.Records[4], "m6")
	})
}

func TestReporter_Report_NonExisting(t *testing.T) {
	reporter := NewEmptyReporter()
	srv := httptest.NewServer(reporter.HttpHandler())

	reporter.ReportOk("t1", "")
	reporter.ReportError("t1", "")
	stat := readStatus(srv.URL)

	assert.Equal(t, Initializing, stat.Status)
	assert.Empty(t, stat.SDKs)
	assert.Equal(t, Initializing, stat.Cache.Status)
	assert.Equal(t, 0, len(stat.Cache.Records))

	reporter.RegisterSdk("t2", &config.SDKConfig{})
	reporter.ReportOk("t1", "")
	reporter.ReportError("t1", "")
	reporter.ReportOk("t2", "")
	stat = readStatus(srv.URL)

	assert.Equal(t, Healthy, stat.Status)
	assert.Equal(t, Healthy, stat.SDKs["t2"].Source.Status)
	assert.Equal(t, Online, stat.SDKs["t2"].Mode)
	assert.Equal(t, 1, len(stat.SDKs["t2"].Source.Records))
	assert.Equal(t, RemoteSrc, stat.SDKs["t2"].Source.Type)
	assert.Equal(t, NA, stat.Cache.Status)
	assert.Equal(t, 0, len(stat.Cache.Records))
}

func TestReporter_Key_Obfuscation(t *testing.T) {
	reporter := NewEmptyReporter()
	srv := httptest.NewServer(reporter.HttpHandler())

	reporter.RegisterSdk("t", &config.SDKConfig{Key: "XxPbCKmzIUGORk4vsufpzw/iC_KABprDEueeQs3yovVnQ"})
	stat := readStatus(srv.URL)

	assert.Equal(t, "****************************************ovVnQ", stat.SDKs["t"].SdkKey)
}

func TestReporter_Offline(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		reporter := NewEmptyReporter()
		reporter.RegisterSdk("t", &config.SDKConfig{Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: "test"}}})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportOk("t", "")
		stat := readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDKs["t"].Source.Status)
		assert.Equal(t, Offline, stat.SDKs["t"].Mode)
		assert.Equal(t, 1, len(stat.SDKs["t"].Source.Records))
		assert.Equal(t, FileSrc, stat.SDKs["t"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("cache invalid", func(t *testing.T) {
		reporter := NewEmptyReporter()
		reporter.RegisterSdk("t", &config.SDKConfig{Offline: config.OfflineConfig{Enabled: true, UseCache: true}})
		srv := httptest.NewServer(reporter.HttpHandler())
		stat := readStatus(srv.URL)

		assert.Equal(t, Down, stat.Status)
		assert.Equal(t, Down, stat.SDKs["t"].Source.Status)
		assert.Equal(t, Offline, stat.SDKs["t"].Mode)
		assert.Equal(t, 1, len(stat.SDKs["t"].Source.Records))
		assert.Equal(t, CacheSrc, stat.SDKs["t"].Source.Type)
		assert.Equal(t, NA, stat.Cache.Status)
		assert.Equal(t, 0, len(stat.Cache.Records))
	})
	t.Run("cache err", func(t *testing.T) {
		reporter := NewReporter(&config.CacheConfig{Redis: config.RedisConfig{Enabled: true}})
		reporter.RegisterSdk("t", &config.SDKConfig{Offline: config.OfflineConfig{Enabled: true, UseCache: true}})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportError("t", "")
		reporter.ReportError("t", "")
		stat := readStatus(srv.URL)

		assert.Equal(t, Down, stat.Status)
		assert.Equal(t, Down, stat.SDKs["t"].Source.Status)
		assert.Equal(t, Offline, stat.SDKs["t"].Mode)
		assert.Equal(t, 2, len(stat.SDKs["t"].Source.Records))
		assert.Equal(t, CacheSrc, stat.SDKs["t"].Source.Type)
	})
	t.Run("cache valid", func(t *testing.T) {
		reporter := NewReporter(&config.CacheConfig{Redis: config.RedisConfig{Enabled: true}})
		reporter.RegisterSdk("t", &config.SDKConfig{Offline: config.OfflineConfig{Enabled: true, UseCache: true}})
		srv := httptest.NewServer(reporter.HttpHandler())
		reporter.ReportOk("t", "")
		reporter.ReportOk(Cache, "")
		stat := readStatus(srv.URL)

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDKs["t"].Source.Status)
		assert.Equal(t, Offline, stat.SDKs["t"].Mode)
		assert.Equal(t, 1, len(stat.SDKs["t"].Source.Records))
		assert.Equal(t, CacheSrc, stat.SDKs["t"].Source.Type)
		assert.Equal(t, Healthy, stat.Cache.Status)
		assert.Equal(t, 1, len(stat.Cache.Records))
	})
}

func TestReporter_Degraded_Calc(t *testing.T) {
	t.Run("1 record first, 1 error", func(t *testing.T) {
		reporter := NewEmptyReporter().(*reporter)
		reporter.RegisterSdk("t", &config.SDKConfig{})
		reporter.ReportError("t", "")
		stat := reporter.GetStatus()

		assert.Equal(t, Down, stat.Status)
		assert.Equal(t, Down, stat.SDKs["t"].Source.Status)
	})
	t.Run("2 records, 1 error then 1 ok", func(t *testing.T) {
		reporter := NewEmptyReporter().(*reporter)
		reporter.RegisterSdk("t", &config.SDKConfig{})
		reporter.ReportError("t", "")
		reporter.ReportOk("t", "")
		stat := reporter.GetStatus()

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDKs["t"].Source.Status)
	})
	t.Run("2 records, 1 ok then 1 error", func(t *testing.T) {
		reporter := NewEmptyReporter().(*reporter)
		reporter.RegisterSdk("t", &config.SDKConfig{})
		reporter.ReportOk("t", "")
		reporter.ReportError("t", "")
		stat := reporter.GetStatus()

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDKs["t"].Source.Status)
	})
	t.Run("3 records, 1 ok then 2 errors", func(t *testing.T) {
		reporter := NewEmptyReporter().(*reporter)
		reporter.RegisterSdk("t", &config.SDKConfig{})
		reporter.ReportOk("t", "")
		reporter.ReportError("t", "")
		reporter.ReportError("t", "")
		stat := reporter.GetStatus()

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Degraded, stat.SDKs["t"].Source.Status)
	})
	t.Run("3 records, 1 ok then 1 error then 1 ok", func(t *testing.T) {
		reporter := NewEmptyReporter().(*reporter)
		reporter.RegisterSdk("t", &config.SDKConfig{})
		reporter.ReportOk("t", "")
		reporter.ReportError("t", "")
		reporter.ReportOk("t", "")
		stat := reporter.GetStatus()

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDKs["t"].Source.Status)
	})
	t.Run("3 records, 1 error then 1 ok then 1 error", func(t *testing.T) {
		reporter := NewEmptyReporter().(*reporter)
		reporter.RegisterSdk("t", &config.SDKConfig{})
		reporter.ReportError("t", "")
		reporter.ReportOk("t", "")
		reporter.ReportError("t", "")
		stat := reporter.GetStatus()

		assert.Equal(t, Healthy, stat.Status)
		assert.Equal(t, Healthy, stat.SDKs["t"].Source.Status)
	})
	t.Run("2 envs 1 down", func(t *testing.T) {
		reporter := NewEmptyReporter().(*reporter)
		reporter.RegisterSdk("t1", &config.SDKConfig{})
		reporter.RegisterSdk("t2", &config.SDKConfig{})
		reporter.ReportError("t1", "")
		reporter.ReportOk("t2", "")
		stat := reporter.GetStatus()

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Down, stat.SDKs["t1"].Source.Status)
		assert.Equal(t, Healthy, stat.SDKs["t2"].Source.Status)
	})
	t.Run("2 envs 1 degraded", func(t *testing.T) {
		reporter := NewEmptyReporter().(*reporter)
		reporter.RegisterSdk("t1", &config.SDKConfig{})
		reporter.RegisterSdk("t2", &config.SDKConfig{})
		reporter.ReportError("t1", "")
		reporter.ReportOk("t1", "")
		reporter.ReportError("t1", "")
		reporter.ReportError("t1", "")
		reporter.ReportOk("t2", "")
		stat := reporter.GetStatus()

		assert.Equal(t, Degraded, stat.Status)
		assert.Equal(t, Degraded, stat.SDKs["t1"].Source.Status)
		assert.Equal(t, Healthy, stat.SDKs["t2"].Source.Status)
	})
}

func TestNewNullReporter(t *testing.T) {
	rep := NewEmptyReporter().(*reporter)
	assert.Empty(t, rep.records)
	assert.Empty(t, rep.GetStatus().SDKs)
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
