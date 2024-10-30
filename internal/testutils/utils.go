package testutils

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/http/httptest"
	"testing"
)

func NewTestRegistrar(conf *config.SDKConfig, cache configcat.ConfigCache) sdk.Registrar {
	return NewTestRegistrarWithStatusReporter(conf, cache, status.NewEmptyReporter())
}

func NewTestRegistrarWithStatusReporter(conf *config.SDKConfig, cache configcat.ConfigCache, reporter status.Reporter) sdk.Registrar {
	ctx := NewTestSdkContext(conf, cache)
	reg, _ := sdk.NewRegistrar(&config.Config{
		SDKs: map[string]*config.SDKConfig{"test": conf},
	}, ctx.MetricsReporter, reporter, cache, log.NewNullLogger())
	return reg
}

func NewTestRegistrarT(t *testing.T) (sdk.Registrar, *configcattest.Handler, string) {
	return NewTestRegistrarTWithStatusReporter(t, status.NewEmptyReporter())
}

func NewTestRegistrarTWithStatusReporter(t *testing.T, reporter status.Reporter) (sdk.Registrar, *configcattest.Handler, string) {
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

func NewTestRegistrarTWithErrorServer(t *testing.T) sdk.Registrar {
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

func NewTestSdkClient(t *testing.T) (map[string]sdk.Client, *configcattest.Handler, string) {
	reg, h, k := NewTestRegistrarT(t)
	return reg.GetAll(), h, k
}

func NewTestSdkContext(conf *config.SDKConfig, cache configcat.ConfigCache) *sdk.Context {
	return &sdk.Context{
		SDKConf:        conf,
		ProxyConf:      &config.HttpProxyConfig{},
		StatusReporter: status.NewEmptyReporter(),
		SdkId:          "test",
		ExternalCache:  cache,
	}
}

func AddSdkIdContextParam(r *http.Request) {
	params := httprouter.Params{httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)
	*r = *r.WithContext(ctx)
}

func AddSdkIdContextParamWithSdkId(r *http.Request, sdkId string) {
	params := httprouter.Params{httprouter.Param{Key: "sdkId", Value: sdkId}}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)
	*r = *r.WithContext(ctx)
}
