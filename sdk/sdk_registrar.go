package sdk

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/metrics"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
)

type Registrar interface {
	GetSdkOrNil(sdkId string) Client
	GetSdkByKeyOrNil(sdkKey string) Client
	GetAll() map[string]Client
	Close()
}

type manualRegistrar struct {
	sdkClients map[string]Client
}

func NewRegistrar(conf *config.Config, metricsReporter metrics.Reporter, statusReporter status.Reporter, externalCache store.Cache, log log.Logger) (Registrar, error) {
	if conf.Profile.IsSet() {
		return newAutoRegistrar(conf, metricsReporter, statusReporter, externalCache, log)
	} else {
		return newManualRegistrar(conf, metricsReporter, statusReporter, externalCache, log)
	}
}

func newManualRegistrar(conf *config.Config, metricsReporter metrics.Reporter, statusReporter status.Reporter, externalCache store.Cache, log log.Logger) (*manualRegistrar, error) {
	regLog := log.WithPrefix("sdk-registrar").WithLevel(conf.Profile.Log.GetLevel())
	sdkClients := make(map[string]Client, len(conf.SDKs))
	for key, sdkConf := range conf.SDKs {
		statusReporter.RegisterSdk(key, sdkConf)
		sdkClients[key] = NewClient(&Context{
			SDKConf:            sdkConf,
			MetricsReporter:    metricsReporter,
			StatusReporter:     statusReporter,
			ProxyConf:          &conf.HttpProxy,
			GlobalDefaultAttrs: conf.DefaultAttrs,
			SdkId:              key,
			ExternalCache:      externalCache,
		}, regLog)
	}
	return &manualRegistrar{sdkClients: sdkClients}, nil
}

func (r *manualRegistrar) GetSdkOrNil(sdkId string) Client {
	return r.sdkClients[sdkId]
}

func (r *manualRegistrar) GetSdkByKeyOrNil(sdkKey string) Client {
	for _, sdkClient := range r.sdkClients {
		key1, key2 := sdkClient.SdkKeys()
		if key1 == sdkKey || (key2 != nil && len(*key2) > 0 && *key2 == sdkKey) {
			return sdkClient
		}
	}
	return nil
}

func (r *manualRegistrar) GetAll() map[string]Client {
	return r.sdkClients
}

func (r *manualRegistrar) Close() {
	for _, sdkClient := range r.sdkClients {
		sdkClient.Close()
	}
}
