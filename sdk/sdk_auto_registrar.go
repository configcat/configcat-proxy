package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/metrics"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/pubsub"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/puzpuzpuz/xsync/v3"
	"io"
	"net/http"
	"time"
)

type AutoRegistrar interface {
	pubsub.SubscriptionHandler[string]
	Registrar
}

type proxyConfigModel struct {
	SDKs    map[string]*sdkConfigModel
	Options optionsModel
}

type optionsModel struct {
	PollInterval   int
	BaseUrl        string
	DataGovernance string
}

type sdkConfigModel struct {
	SDKKey string
}

type autoRegistrar struct {
	sdkClients      *xsync.MapOf[string, Client]
	httpClient      *http.Client
	ctx             context.Context
	ctxCancel       func()
	conf            *config.Config
	metricsReporter metrics.Reporter
	statusReporter  status.Reporter
	externalCache   configcat.ConfigCache
	log             log.Logger
	poller          *time.Ticker
	pubsub.Publisher[string]
}

func newAutoRegistrar(conf *config.Config, metricsReporter metrics.Reporter, statusReporter status.Reporter, externalCache configcat.ConfigCache, log log.Logger) (*autoRegistrar, error) {
	regLog := log.WithPrefix("auto-sdk-registrar").WithLevel(conf.AutoSDK.Log.GetLevel())
	ctx, cancel := context.WithCancel(context.Background())
	registrar := &autoRegistrar{
		conf:            conf,
		sdkClients:      xsync.NewMapOf[string, Client](),
		metricsReporter: metricsReporter,
		statusReporter:  statusReporter,
		externalCache:   externalCache,
		log:             regLog,
		Publisher:       pubsub.NewPublisher[string](),
		httpClient:      http.DefaultClient,
		ctx:             ctx,
		ctxCancel:       cancel,
	}

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, time.Second*15)
	defer timeoutCancel()
	autoConfig, err := registrar.fetchConfig(timeoutCtx)
	if err != nil {
		regLog.Errorf("%v", err)
		return nil, err
	}
	for sdkId, sdkModel := range autoConfig.SDKs {
		sdkConfig := registrar.buildSdkConfig(sdkId, sdkModel, &autoConfig.Options)
		statusReporter.RegisterSdk(sdkId, sdkConfig)
		registrar.sdkClients.Store(sdkId, registrar.buildSdkClient(sdkId, sdkConfig))
	}
	registrar.poller = time.NewTicker(time.Duration(conf.AutoSDK.PollInterval) * time.Second)
	go registrar.run()

	return registrar, nil
}

func (r *autoRegistrar) GetSdkOrNil(sdkId string) Client {
	sdk, _ := r.sdkClients.Load(sdkId)
	return sdk
}

func (r *autoRegistrar) GetAll() map[string]Client {
	all := make(map[string]Client, r.sdkClients.Size())
	r.sdkClients.Range(func(key string, value Client) bool {
		all[key] = value
		return true
	})
	return all
}

func (r *autoRegistrar) Close() {
	r.ctxCancel()
	r.Publisher.Close()
	if r.poller != nil {
		r.poller.Stop()
	}
	r.sdkClients.Range(func(key string, value Client) bool {
		value.Close()
		r.sdkClients.Delete(key)
		return true
	})
	r.log.Reportf("shutdown complete")
}

func (r *autoRegistrar) run() {
	for {
		select {
		case <-r.poller.C:
			autoConfig, err := r.fetchConfig(r.ctx)
			if err != nil {
				r.log.Errorf("%v", err)
			} else {
				existingKeys := utils.KeysOfSyncMap(r.sdkClients)
				remoteKeys := utils.KeysOfMap(autoConfig.SDKs)

				toAddKeys := utils.Except(remoteKeys, existingKeys)
				toDeleteKeys := utils.Except(existingKeys, remoteKeys)

				r.deleteSdkClients(toDeleteKeys)
				r.addSdkClients(toAddKeys, autoConfig)
			}
		case <-r.ctx.Done():
			return
		}
	}
}

func (r *autoRegistrar) fetchConfig(ctx context.Context) (*proxyConfigModel, error) {
	url := r.conf.AutoSDK.BaseUrl + "/v1/proxy/config"
	r.log.Debugf("fetching remote configuration from %s?key=%s", url, r.conf.AutoSDK.Key)
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error during fetching remote configuration: %v", err)
	}
	query := request.URL.Query()
	query.Add("key", r.conf.AutoSDK.Key)
	request.URL.RawQuery = query.Encode()
	response, err := r.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error during fetching remote configuration: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error during fetching remote configuration, status code: %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error during reading configuration response: %v", err)
	}
	parsedResponse := proxyConfigModel{}
	if err := json.Unmarshal(body, &parsedResponse); err != nil {
		return nil, fmt.Errorf("error during parsing configuration response: %v", err)
	}
	r.log.Debugf("remote configuration fetch was successful, got %d SDK keys", len(parsedResponse.SDKs))
	return &parsedResponse, nil
}

func (r *autoRegistrar) buildSdkConfig(sdkId string, sdkModel *sdkConfigModel, opts *optionsModel) *config.SDKConfig {
	sdkConfig := &config.SDKConfig{
		BaseUrl:        opts.BaseUrl,
		Key:            sdkModel.SDKKey,
		PollInterval:   opts.PollInterval,
		DataGovernance: opts.DataGovernance,
		Log:            r.conf.AutoSDK.Log,
	}
	if localSdkConfig, ok := r.conf.SDKs[sdkId]; ok {
		sdkConfig.DefaultAttrs = localSdkConfig.DefaultAttrs
		sdkConfig.Offline = localSdkConfig.Offline
		sdkConfig.Log = localSdkConfig.Log
	}
	return sdkConfig
}

func (r *autoRegistrar) buildSdkClient(sdkId string, sdkConfig *config.SDKConfig) Client {
	return NewClient(&Context{
		SDKConf:            sdkConfig,
		MetricsReporter:    r.metricsReporter,
		StatusReporter:     r.statusReporter,
		ProxyConf:          &r.conf.HttpProxy,
		GlobalDefaultAttrs: r.conf.DefaultAttrs,
		SdkId:              sdkId,
		ExternalCache:      r.externalCache,
	}, r.log)
}

func (r *autoRegistrar) deleteSdkClients(sdkIds []string) {
	if len(sdkIds) == 0 {
		r.log.Debugf("no SDK clients to remove")
		return
	}

	r.log.Debugf("removing %d SDK clients", len(sdkIds))
	for _, sdkId := range sdkIds {
		if sdkClient, ok := r.sdkClients.LoadAndDelete(sdkId); ok {
			sdkClient.Close()
			r.statusReporter.RemoveSdk(sdkId)
			r.Publish(sdkId)
		}
	}
}

func (r *autoRegistrar) addSdkClients(sdkIds []string, config *proxyConfigModel) {
	if len(sdkIds) == 0 {
		r.log.Debugf("no SDK clients to add")
		return
	}

	r.log.Debugf("adding %d SDK clients", len(sdkIds))
	for _, sdkId := range sdkIds {
		if sdkModel, ok := config.SDKs[sdkId]; ok {
			sdkConfig := r.buildSdkConfig(sdkId, sdkModel, &config.Options)
			if _, loaded := r.sdkClients.LoadOrCompute(sdkId, func() Client {
				return r.buildSdkClient(sdkId, sdkConfig)
			}); !loaded {
				r.statusReporter.RegisterSdk(sdkId, sdkConfig)
				r.Publish(sdkId)
			}
		}
	}
}
