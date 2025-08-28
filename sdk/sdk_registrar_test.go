package sdk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/stretchr/testify/assert"
)

func TestRegistrar_GetSdkOrNil(t *testing.T) {
	reg, _ := NewRegistrar(&config.Config{
		SDKs: map[string]*config.SDKConfig{"test": {Key: "key"}},
	}, nil, status.NewEmptyReporter(), nil, log.NewNullLogger())
	defer reg.Close()

	assert.NotNil(t, reg.GetSdkOrNil("test"))
}

func TestRegistrar_GetSdkByKeyOrNil(t *testing.T) {
	reg, _ := NewRegistrar(&config.Config{
		SDKs: map[string]*config.SDKConfig{"test": {Key: "key"}},
	}, nil, status.NewEmptyReporter(), nil, log.NewNullLogger())
	defer reg.Close()

	assert.NotNil(t, reg.GetSdkByKeyOrNil("key"))
}

func TestRegistrar_All(t *testing.T) {
	reg, _ := NewRegistrar(&config.Config{
		SDKs: map[string]*config.SDKConfig{"test1": {Key: "key1"}, "test2": {Key: "key2"}},
	}, nil, status.NewEmptyReporter(), nil, log.NewNullLogger())
	defer reg.Close()

	assert.Equal(t, 2, len(reg.GetAll()))
}

func TestRegistrar_StatusReporter(t *testing.T) {
	reporter := status.NewEmptyReporter()
	reg, _ := NewRegistrar(&config.Config{
		SDKs: map[string]*config.SDKConfig{"test1": {Key: "key1"}},
	}, nil, reporter, nil, log.NewNullLogger())
	defer reg.Close()

	assert.NotEmpty(t, reporter.GetStatus().SDKs)
}

func TestClient_Close(t *testing.T) {
	reg, _ := NewRegistrar(&config.Config{
		SDKs: map[string]*config.SDKConfig{"test": {Key: "key"}},
	}, nil, status.NewEmptyReporter(), nil, log.NewNullLogger())

	c := reg.GetSdkOrNil("test").(*client)
	reg.Close()
	testutils.WithTimeout(1*time.Second, func() {
		<-c.ctx.Done()
	})
}

func TestNewRegistrar(t *testing.T) {
	cache := miniredis.RunT(t)
	extCache := newRedisCache(cache.Addr())
	autConfigCacheJson, _ := json.Marshal(model.ProxyConfigModel{
		SDKs: map[string]*model.SdkConfigModel{"test": {Key1: configcattest.RandomSDKKey()}},
	})
	autConfigCacheEntry := cacheSegmentsToBytes("etag", autConfigCacheJson)
	_ = cache.Set("configcat-proxy-profile-test-reg", string(autConfigCacheEntry))
	reg, _ := NewRegistrar(&config.Config{
		Profile: config.ProfileConfig{Key: "test-reg", Secret: "secret", PollInterval: 60},
	}, nil, status.NewEmptyReporter(), extCache, log.NewDebugLogger())
	defer reg.Close()
	assert.IsType(t, &autoRegistrar{}, reg)
}

func TestBuildProxy(t *testing.T) {
	proxy := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok from proxy"))
		}))
	defer proxy.Close()
	transport := buildTransport(&config.HttpProxyConfig{Url: proxy.URL}, log.NewNullLogger())
	client := &http.Client{
		Transport: transport,
	}
	rsp, err := client.Get("http://nonexisting")
	assert.NoError(t, err)

	body, err := io.ReadAll(rsp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "ok from proxy", string(body))
}
