package testutils

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcattest"
	"net/http/httptest"
	"testing"
)

func NewTestSdkClient(t *testing.T) (map[string]sdk.Client, *configcattest.Handler, string) {
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
	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	ctx := NewTestSdkContext(&opts, &config.CacheConfig{})
	client := sdk.NewClient(ctx, log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return map[string]sdk.Client{"test": client}, &h, key
}

func NewTestSdkContext(conf *config.SDKConfig, cacheConf *config.CacheConfig) *sdk.Context {
	if cacheConf == nil {
		cacheConf = &config.CacheConfig{}
	}
	return &sdk.Context{
		SDKConf:        conf,
		ProxyConf:      &config.HttpProxyConfig{},
		CacheConf:      cacheConf,
		StatusReporter: status.NewNullReporter(),
		MetricsHandler: nil,
		EvalReporter:   nil,
		SdkId:          "test",
	}
}
