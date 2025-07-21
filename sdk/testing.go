//go:build testing

package sdk

import (
	"encoding/json"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk/store"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcattest"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func NewTestRegistrar(conf *config.SDKConfig, cache store.Cache) Registrar {
	return NewTestRegistrarWithStatusReporter(conf, cache, status.NewEmptyReporter())
}

func NewTestRegistrarWithStatusReporter(conf *config.SDKConfig, cache store.Cache, reporter status.Reporter) Registrar {
	ctx := NewTestSdkContext(conf, cache)
	reg, _ := NewRegistrar(&config.Config{
		SDKs: map[string]*config.SDKConfig{"test": conf},
	}, ctx.MetricsReporter, reporter, cache, log.NewNullLogger())
	return reg
}

func NewTestRegistrarT(t *testing.T) (Registrar, *configcattest.Handler, string) {
	return NewTestRegistrarTWithStatusReporter(t, status.NewEmptyReporter())
}

func NewTestRegistrarTWithStatusReporter(t *testing.T, reporter status.Reporter) (Registrar, *configcattest.Handler, string) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
			Rules: []configcattest.Rule{
				{
					Comparator:          configcat.OpEq,
					Value:               false,
					ComparisonValue:     "test",
					ComparisonAttribute: "Identifier",
				},
			},
		},
	})
	srv := httptest.NewServer(&h)
	reg := NewTestRegistrarWithStatusReporter(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil, reporter)
	t.Cleanup(func() {
		srv.Close()
		reg.Close()
	})
	return reg, &h, key
}

func NewTestRegistrarTWithErrorServer(t *testing.T) Registrar {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	srv := httptest.NewServer(&h)
	reg := NewTestRegistrarWithStatusReporter(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil, status.NewEmptyReporter())
	t.Cleanup(func() {
		srv.Close()
		reg.Close()
	})
	return reg
}

func NewTestSdkClient(t *testing.T) (map[string]Client, *configcattest.Handler, string) {
	reg, h, k := NewTestRegistrarT(t)
	return reg.GetAll(), h, k
}

func NewTestSdkContext(conf *config.SDKConfig, cache store.Cache) *Context {
	return &Context{
		SDKConf:        conf,
		ProxyConf:      &config.HttpProxyConfig{},
		StatusReporter: status.NewEmptyReporter(),
		SdkId:          "test",
		ExternalCache:  cache,
	}
}

func NewTestAutoRegistrarWithAutoConfig(t *testing.T, autoConf config.AutoSDKConfig, logger log.Logger) (AutoRegistrar, *TestSdkRegistrarHandler, string) {
	return NewTestAutoRegistrar(t, config.Config{AutoSDK: autoConf}, nil, logger)
}

func NewTestAutoRegistrarWithCache(t *testing.T, cachePoll int, cache store.Cache, logger log.Logger) AutoRegistrar {
	conf := config.Config{AutoSDK: config.AutoSDKConfig{Key: "test-reg", PollInterval: 60}, GlobalOfflineConfig: config.GlobalOfflineConfig{
		CachePollInterval: cachePoll,
		Enabled:           true,
	}}
	reg, _ := newAutoRegistrar(&conf, nil, status.NewEmptyReporter(), cache, logger)
	t.Cleanup(reg.Close)
	return reg
}

func NewTestAutoRegistrar(t *testing.T, conf config.Config, cache store.Cache, logger log.Logger) (AutoRegistrar, *TestSdkRegistrarHandler, string) {
	sdkKey := configcattest.RandomSDKKey()
	var sdkHandler configcattest.Handler
	_ = sdkHandler.SetFlags(sdkKey, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	sdkSrv := httptest.NewServer(&sdkHandler)
	h := TestSdkRegistrarHandler{result: model.ProxyConfigModel{
		Options: model.OptionsModel{
			PollInterval:   60,
			DataGovernance: "global",
		},
		SDKs: map[string]*model.SdkConfigModel{"test": {SDKKey: sdkKey}},
	}, sdkHandler: &sdkHandler}
	configSrv := httptest.NewServer(&h)

	conf.AutoSDK.SdkBaseUrl = sdkSrv.URL
	conf.AutoSDK.BaseUrl = configSrv.URL
	reg, _ := newAutoRegistrar(&conf, nil, status.NewEmptyReporter(), cache, logger)
	t.Cleanup(func() {
		sdkSrv.Close()
		configSrv.Close()
		reg.Close()
	})
	return reg, &h, sdkKey
}

type TestSdkRegistrarHandler struct {
	mu         sync.RWMutex
	result     model.ProxyConfigModel
	sdkHandler *configcattest.Handler
}

func (h *TestSdkRegistrarHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "only GET is allowed", http.StatusMethodNotAllowed)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	body, _ := json.Marshal(h.result)
	etag := utils.GenerateEtag(body)
	w.Header().Set("ETag", etag)
	receivedEtag := req.Header.Get("If-None-Match")
	if receivedEtag == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *TestSdkRegistrarHandler) AddSdk(sdkId string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sdkKey := configcattest.RandomSDKKey()
	_ = h.sdkHandler.SetFlags(sdkKey, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	h.result.SDKs[sdkId] = &model.SdkConfigModel{SDKKey: sdkKey}
}

func (h *TestSdkRegistrarHandler) RemoveSdk(sdkId string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.result.SDKs, sdkId)
}

func (h *TestSdkRegistrarHandler) ModifyGlobalOpts(optionsModel model.OptionsModel) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.result.Options = optionsModel
}
