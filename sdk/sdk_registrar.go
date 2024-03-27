package sdk

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/metrics"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/log"
	configcat "github.com/configcat/go-sdk/v9"
)

type Registrar interface {
	GetSdkOrNil(sdkId string) Client
	GetAll() map[string]Client
	Close()
}

type registrar struct {
	sdkClients map[string]Client
}

func NewRegistrar(conf *config.Config, metricsReporter metrics.Reporter, statusReporter status.Reporter, externalCache configcat.ConfigCache, log log.Logger) Registrar {
	sdkClients := make(map[string]Client, len(conf.SDKs))
	for key, sdkConf := range conf.SDKs {
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
	return &registrar{sdkClients: sdkClients}
}

func (r *registrar) GetSdkOrNil(sdkId string) Client {
	return r.sdkClients[sdkId]
}

func (r *registrar) GetAll() map[string]Client {
	return r.sdkClients
}

func (r *registrar) Close() {
	for _, sdkClient := range r.sdkClients {
		sdkClient.Close()
	}
}
