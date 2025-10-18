package sdk

import (
	"net/http"
	"net/url"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
)

type Registrar interface {
	GetSdkOrNil(sdkId string) Client
	GetSdkByKeyOrNil(sdkKey string) Client
	RefreshAll()
	GetAll() map[string]Client
	Close()
}

type manualRegistrar struct {
	sdkClients         map[string]Client
	sdkClientsBySdkKey map[string]Client
}

func NewRegistrar(conf *config.Config, telemetryReporter telemetry.Reporter, statusReporter status.Reporter, externalCache store.Cache, log log.Logger) (Registrar, error) {
	if conf.Profile.IsSet() {
		return newAutoRegistrar(conf, telemetryReporter, statusReporter, externalCache, log)
	} else {
		return newManualRegistrar(conf, telemetryReporter, statusReporter, externalCache, log)
	}
}

func newManualRegistrar(conf *config.Config, telemetryReporter telemetry.Reporter, statusReporter status.Reporter, externalCache store.Cache, log log.Logger) (*manualRegistrar, error) {
	regLog := log.WithPrefix("sdk-registrar").WithLevel(conf.Profile.Log.GetLevel())
	transport := buildTransport(&conf.HttpProxy, regLog)
	sdkClients := make(map[string]Client, len(conf.SDKs))
	sdkClientsBySdkKey := make(map[string]Client, len(conf.SDKs))
	for key, sdkConf := range conf.SDKs {
		statusReporter.RegisterSdk(key, sdkConf)
		sdkClient := NewClient(&Context{
			SDKConf:            sdkConf,
			TelemetryReporter:  telemetryReporter,
			StatusReporter:     statusReporter,
			GlobalDefaultAttrs: conf.DefaultAttrs,
			SdkId:              key,
			ExternalCache:      externalCache,
			Transport:          transport,
		}, regLog)
		sdkClients[key] = sdkClient
		sdkClientsBySdkKey[sdkConf.Key] = sdkClient
	}
	return &manualRegistrar{sdkClients: sdkClients, sdkClientsBySdkKey: sdkClientsBySdkKey}, nil
}

func (r *manualRegistrar) GetSdkOrNil(id string) Client {
	return r.sdkClients[id]
}

func (r *manualRegistrar) GetSdkByKeyOrNil(sdkKey string) Client {
	return r.sdkClientsBySdkKey[sdkKey]
}

func (r *manualRegistrar) RefreshAll() {
	for _, sdkClient := range r.sdkClients {
		_ = sdkClient.Refresh()
	}
}

func (r *manualRegistrar) GetAll() map[string]Client {
	return r.sdkClients
}

func (r *manualRegistrar) Close() {
	for _, sdkClient := range r.sdkClients {
		sdkClient.Close()
	}
}

func buildTransport(proxyConf *config.HttpProxyConfig, log log.Logger) http.RoundTripper {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyConf.Url != "" {
		proxyUrl, err := url.Parse(proxyConf.Url)
		if err != nil {
			log.Errorf("failed to parse proxy url: %s", proxyConf.Url)
		} else {
			transport.Proxy = http.ProxyURL(proxyUrl)
			log.Reportf("using HTTP proxy: %s", proxyConf.Url)
		}
	}
	return OverrideUserAgent(transport)
}
