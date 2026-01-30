//go:build testing

package sdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/configcat/configcat-proxy/cache"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcattest"
)

func NewTestRegistrar(conf *config.SDKConfig, cache cache.ReaderWriter) Registrar {
	return NewTestRegistrarWithStatusReporter(conf, cache, status.NewEmptyReporter())
}

func NewTestRegistrarWithStatusReporter(conf *config.SDKConfig, cache cache.ReaderWriter, reporter status.Reporter) Registrar {
	ctx := NewTestSdkContext(conf, cache)
	reg, _ := NewRegistrar(&config.Config{
		SDKs: map[string]*config.SDKConfig{"test": conf},
	}, ctx.TelemetryReporter, reporter, cache, log.NewNullLogger())
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

func NewTestSdkContext(conf *config.SDKConfig, cache cache.ReaderWriter) *Context {
	return &Context{
		SDKConf:           conf,
		Transport:         http.DefaultTransport,
		StatusReporter:    status.NewEmptyReporter(),
		TelemetryReporter: telemetry.NewEmptyReporter(),
		SdkId:             "test",
		ExternalCache:     cache,
	}
}

func NewTestAutoRegistrarWithAutoConfig(t *testing.T, autoConf config.ProfileConfig, logger log.Logger) (AutoRegistrar, *TestSdkRegistrarHandler, string) {
	return NewTestAutoRegistrar(t, config.Config{Profile: autoConf}, nil, logger)
}

func NewTestAutoRegistrarWithCache(t *testing.T, cachePoll int, cache cache.ReaderWriter, logger log.Logger) AutoRegistrar {
	conf := config.Config{Profile: config.ProfileConfig{Key: "test-reg", PollInterval: 60}, GlobalOfflineConfig: config.GlobalOfflineConfig{
		CachePollInterval: cachePoll,
		Enabled:           true,
	}}
	reg, _ := newAutoRegistrar(&conf, telemetry.NewEmptyReporter(), status.NewEmptyReporter(), cache, logger)
	t.Cleanup(reg.Close)
	return reg
}

func NewTestAutoRegistrar(t *testing.T, conf config.Config, cache cache.ReaderWriter, logger log.Logger) (AutoRegistrar, *TestSdkRegistrarHandler, string) {
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
		SDKs: map[string]*model.SdkConfigModel{"test": {Key1: sdkKey}},
	}, sdkHandler: &sdkHandler}
	configSrv := httptest.NewServer(&h)

	conf.Profile.SDKs.BaseUrl = sdkSrv.URL
	conf.Profile.BaseUrl = configSrv.URL
	reg, _ := newAutoRegistrar(&conf, telemetry.NewEmptyReporter(), status.NewEmptyReporter(), cache, logger)
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

func (h *TestSdkRegistrarHandler) AddSdk(sdkId string) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	sdkKey := configcattest.RandomSDKKey()
	_ = h.sdkHandler.SetFlags(sdkKey, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	h.result.SDKs[sdkId] = &model.SdkConfigModel{Key1: sdkKey}
	return sdkKey
}

func (h *TestSdkRegistrarHandler) RotateSdkKey(sdkId string) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	sdkKey := configcattest.RandomSDKKey()
	h.result.SDKs[sdkId].Key2 = sdkKey
	return sdkKey
}

func (h *TestSdkRegistrarHandler) RemoveSdkKey(sdkId string, primary bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if primary {
		h.result.SDKs[sdkId].Key1 = h.result.SDKs[sdkId].Key2
		h.result.SDKs[sdkId].Key2 = ""
	} else {
		h.result.SDKs[sdkId].Key2 = ""
	}
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
