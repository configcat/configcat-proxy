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
	GetAll() map[string]Client
	Close()
}

type manualRegistrar struct {
	sdkClients map[string]Client
}

func NewRegistrar(conf *config.Config, metricsReporter metrics.Reporter, statusReporter status.Reporter, externalCache store.Cache, log log.Logger) (Registrar, error) {
	if conf.AutoSDK.IsSet() {
		return newAutoRegistrar(conf, metricsReporter, statusReporter, externalCache, log)
	} else {
		return newManualRegistrar(conf, metricsReporter, statusReporter, externalCache, log)
	}
}

func newManualRegistrar(conf *config.Config, metricsReporter metrics.Reporter, statusReporter status.Reporter, externalCache store.Cache, log log.Logger) (*manualRegistrar, error) {
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
		}, log)
	}
	return &manualRegistrar{sdkClients: sdkClients}, nil
}

func (r *manualRegistrar) GetSdkOrNil(sdkId string) Client {
	return r.sdkClients[sdkId]
}

func (r *manualRegistrar) GetAll() map[string]Client {
	return r.sdkClients
}

func (r *manualRegistrar) Close() {
	for _, sdkClient := range r.sdkClients {
		sdkClient.Close()
	}
}
