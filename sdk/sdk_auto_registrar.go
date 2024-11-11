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
	"net/url"
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
	options         *optionsModel
	cacheKey        string
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
	var transport = http.DefaultTransport.(*http.Transport)
	if !conf.GlobalOfflineConfig.Enabled && conf.HttpProxy.Url != "" {
		proxyUrl, err := url.Parse(conf.HttpProxy.Url)
		if err != nil {
			regLog.Errorf("failed to parse proxy url: %s", conf.HttpProxy.Url)
		} else {
			transport.Proxy = http.ProxyURL(proxyUrl)
			regLog.Reportf("using HTTP proxy: %s", conf.HttpProxy.Url)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	registrar := &autoRegistrar{
		conf:            conf,
		sdkClients:      xsync.NewMapOf[string, Client](),
		metricsReporter: metricsReporter,
		statusReporter:  statusReporter,
		externalCache:   externalCache,
		log:             regLog,
		Publisher:       pubsub.NewPublisher[string](),
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		ctx:       ctx,
		ctxCancel: cancel,
		cacheKey:  "configcat-proxy-conf/" + conf.AutoSDK.Key,
	}

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, time.Second*15)
	defer timeoutCancel()
	autoConfig, err := registrar.getConfig(timeoutCtx)
	if err != nil {
		regLog.Errorf("%v", err)
		return nil, err
	}
	registrar.options = &autoConfig.Options
	for sdkId, sdkModel := range autoConfig.SDKs {
		sdkConfig := registrar.buildSdkConfig(sdkId, sdkModel)
		statusReporter.RegisterSdk(sdkId, sdkConfig)
		registrar.sdkClients.Store(sdkId, registrar.buildSdkClient(sdkId, sdkConfig))
	}
	var interval time.Duration
	if conf.GlobalOfflineConfig.Enabled {
		interval = time.Duration(conf.GlobalOfflineConfig.CachePollInterval) * time.Second
	} else {
		interval = time.Duration(conf.AutoSDK.PollInterval) * time.Second
	}
	registrar.poller = time.NewTicker(interval)
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
			autoConfig, err := r.getConfig(r.ctx)
			if err != nil {
				r.log.Errorf("%v", err)
			} else {
				existingKeys := utils.KeysOfSyncMap(r.sdkClients)
				remoteKeys := utils.KeysOfMap(autoConfig.SDKs)

				toAddKeys := utils.Except(remoteKeys, existingKeys)
				toDeleteKeys := utils.Except(existingKeys, remoteKeys)

				r.deleteSdkClients(toDeleteKeys)
				if r.shouldUpdateOptions(&autoConfig.Options) {
					r.options = &autoConfig.Options
					resetKeys := utils.Except(remoteKeys, toAddKeys)
					r.resetSdkClients(resetKeys, autoConfig)
				}
				r.addSdkClients(toAddKeys, autoConfig)
			}
		case <-r.ctx.Done():
			return
		}
	}
}

func (r *autoRegistrar) getConfig(ctx context.Context) (*proxyConfigModel, error) {
	if r.conf.GlobalOfflineConfig.Enabled {
		if r.externalCache == nil {
			return nil, fmt.Errorf("could not load auto sdk configuration: offline mode is enabled without an external cache")
		}
		cached, err := r.externalCache.Get(ctx, r.cacheKey)
		if err != nil {
			return nil, fmt.Errorf("could not read a valid auto sdk configuration from cache: %v", err)
		}
		if len(cached) == 0 {
			return nil, fmt.Errorf("no valid auto sdk configuration found in cache")
		}
		return r.parseConfig(cached)
	}
	fetched, err := r.fetchConfig(ctx)
	if err != nil {
		if r.externalCache == nil {
			return nil, err
		}
		r.log.Errorf("could not fetch auto sdk configuration, falling back to cache: %v", err)
		cached, err := r.externalCache.Get(ctx, r.cacheKey)
		if err != nil {
			return nil, fmt.Errorf("could not read a valid auto sdk configuration from cache: %v", err)
		}
		if len(cached) == 0 {
			return nil, fmt.Errorf("no valid auto sdk configuration found in cache")
		}
		return r.parseConfig(cached)
	}
	if r.externalCache != nil {
		err = r.externalCache.Set(ctx, r.cacheKey, fetched)
		if err != nil {
			r.log.Errorf("could not write the auto sdk configuration to cache: %v", err)
		}
	}
	return r.parseConfig(fetched)
}

func (r *autoRegistrar) parseConfig(body []byte) (*proxyConfigModel, error) {
	parsed := proxyConfigModel{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("error during parsing auto sdk configuration: %v", err)
	}
	r.log.Debugf("auto sdk configuration loaded, got %d SDK keys", len(parsed.SDKs))
	return &parsed, nil
}

func (r *autoRegistrar) fetchConfig(ctx context.Context) ([]byte, error) {
	apiUrl := r.conf.AutoSDK.BaseUrl + "/v1/proxy/config"
	r.log.Debugf("fetching remote configuration from %s?key=%s", apiUrl, r.conf.AutoSDK.Key)
	request, err := http.NewRequestWithContext(ctx, "GET", apiUrl, nil)
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
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error during fetching remote configuration, status code: %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error during reading configuration response: %v", err)
	}
	return body, nil
}

func (r *autoRegistrar) buildSdkConfig(sdkId string, sdkModel *sdkConfigModel) *config.SDKConfig {
	sdkConfig := &config.SDKConfig{
		BaseUrl:        r.options.BaseUrl,
		Key:            sdkModel.SDKKey,
		PollInterval:   r.options.PollInterval,
		DataGovernance: r.options.DataGovernance,
		Log:            r.conf.AutoSDK.Log,
	}
	if localSdkConfig, ok := r.conf.SDKs[sdkId]; ok {
		sdkConfig.DefaultAttrs = localSdkConfig.DefaultAttrs
		sdkConfig.Offline = localSdkConfig.Offline
		sdkConfig.Log = localSdkConfig.Log
	}
	if r.conf.GlobalOfflineConfig.Enabled && !sdkConfig.Offline.Enabled {
		sdkConfig.Offline.Enabled = true
		sdkConfig.Offline.UseCache = true
		sdkConfig.Offline.CachePollInterval = r.conf.GlobalOfflineConfig.CachePollInterval
		sdkConfig.Offline.Log = r.conf.GlobalOfflineConfig.Log
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
			sdkConfig := r.buildSdkConfig(sdkId, sdkModel)
			if _, loaded := r.sdkClients.LoadOrCompute(sdkId, func() Client {
				return r.buildSdkClient(sdkId, sdkConfig)
			}); !loaded {
				r.statusReporter.RegisterSdk(sdkId, sdkConfig)
				r.Publish(sdkId)
			}
		}
	}
}

func (r *autoRegistrar) resetSdkClients(sdkIds []string, config *proxyConfigModel) {
	if len(sdkIds) == 0 {
		r.log.Debugf("no SDK clients to reset")
		return
	}

	r.log.Debugf("resetting %d SDK clients", len(sdkIds))
	for _, sdkId := range sdkIds {
		if sdkModel, ok := config.SDKs[sdkId]; ok {
			sdkConfig := r.buildSdkConfig(sdkId, sdkModel)
			sdkClient := r.buildSdkClient(sdkId, sdkConfig)
			if existing, loaded := r.sdkClients.LoadAndStore(sdkId, sdkClient); loaded {
				existing.Close()
				r.Publish(sdkId)
			}
		}
	}
}

func (r *autoRegistrar) shouldUpdateOptions(options *optionsModel) bool {
	return r.options.BaseUrl != options.BaseUrl ||
		r.options.PollInterval != options.PollInterval ||
		r.options.DataGovernance != options.DataGovernance
}
