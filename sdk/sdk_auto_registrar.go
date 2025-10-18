package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/pubsub"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/puzpuzpuz/xsync/v3"
)

type AutoRegistrar interface {
	Refresh()
	pubsub.SubscriptionHandler[string]
	Registrar
}

type autoRegistrar struct {
	refreshChan        chan struct{}
	options            *model.OptionsModel
	cacheKey           string
	etag               string
	sdkClients         *xsync.MapOf[string, Client]
	sdkClientsBySdkKey *xsync.MapOf[string, Client]
	httpClient         *http.Client
	ctx                context.Context
	ctxCancel          func()
	conf               *config.Config
	telemetryReporter  telemetry.Reporter
	statusReporter     status.Reporter
	cache              store.Cache
	log                log.Logger
	sdkTransport       http.RoundTripper
	pubsub.Publisher[string]
}

func newAutoRegistrar(conf *config.Config, telemetryReporter telemetry.Reporter, statusReporter status.Reporter, cache store.Cache, log log.Logger) (*autoRegistrar, error) {
	regLog := log.WithPrefix("profile-sdk-registrar").WithLevel(conf.Profile.Log.GetLevel())
	transport := buildTransport(&conf.HttpProxy, regLog)
	var profileTransport = telemetryReporter.InstrumentHttpClient(transport, telemetry.NewKV("configcat.source", "profile"))
	registrar := &autoRegistrar{
		refreshChan:        make(chan struct{}),
		conf:               conf,
		sdkClients:         xsync.NewMapOf[string, Client](),
		sdkClientsBySdkKey: xsync.NewMapOf[string, Client](),
		telemetryReporter:  telemetryReporter,
		statusReporter:     statusReporter,
		cache:              cache,
		log:                regLog,
		Publisher:          pubsub.NewPublisher[string](),
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: profileTransport,
		},
		cacheKey:     "configcat-proxy-profile-" + conf.Profile.Key,
		etag:         "initial",
		sdkTransport: transport,
	}
	registrar.ctx, registrar.ctxCancel = context.WithCancel(context.Background())

	timeoutCtx, timeoutCancel := context.WithTimeout(registrar.ctx, time.Second*15)
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
		sdkClient := registrar.buildSdkClient(sdkId, sdkConfig, sdkModel)
		registrar.sdkClients.Store(sdkId, sdkClient)
		if sdkModel.Key1 != "" {
			registrar.sdkClientsBySdkKey.Store(sdkModel.Key1, sdkClient)
		}
		if sdkModel.Key2 != "" {
			registrar.sdkClientsBySdkKey.Store(sdkModel.Key2, sdkClient)
		}
	}
	var interval int
	if conf.GlobalOfflineConfig.Enabled {
		interval = conf.GlobalOfflineConfig.CachePollInterval
	} else {
		interval = conf.Profile.PollInterval
	}
	go registrar.run(interval)
	return registrar, nil
}

func (r *autoRegistrar) GetSdkOrNil(key string) Client {
	if sdk, ok := r.sdkClients.Load(key); ok {
		return sdk
	}
	return nil
}

func (r *autoRegistrar) GetSdkByKeyOrNil(sdkKey string) Client {
	if sdk, ok := r.sdkClientsBySdkKey.Load(sdkKey); ok {
		return sdk
	}
	return nil
}

func (r *autoRegistrar) RefreshAll() {
	r.sdkClients.Range(func(key string, value Client) bool {
		_ = value.Refresh()
		return true
	})
}

func (r *autoRegistrar) GetAll() map[string]Client {
	all := make(map[string]Client, r.sdkClients.Size())
	r.sdkClients.Range(func(key string, value Client) bool {
		all[key] = value
		return true
	})
	return all
}

func (r *autoRegistrar) Refresh() {
	select {
	case <-r.ctx.Done():
		return
	default:
		r.refreshChan <- struct{}{}
	}
}

func (r *autoRegistrar) Close() {
	r.ctxCancel()
	r.Publisher.Close()
	r.sdkClients.Range(func(key string, value Client) bool {
		value.Close()
		r.sdkClients.Delete(key)
		return true
	})
	r.sdkClientsBySdkKey.Clear()
	r.log.Reportf("shutdown complete")
}

func (r *autoRegistrar) run(interval int) {
	inter := interval
	if inter < 1 {
		inter = config.DefaultAutoSdkPollInterval
	}
	poller := time.NewTicker(time.Duration(inter) * time.Second)
	defer poller.Stop()
	for {
		select {
		case <-poller.C:
			r.refreshConfig()
		case <-r.refreshChan:
			r.refreshConfig()
		case <-r.ctx.Done():
			return
		}
	}
}

func (r *autoRegistrar) refreshConfig() {
	autoConfig, err := r.getConfig(r.ctx)
	if err != nil {
		r.log.Errorf("%v", err)
	} else if autoConfig != nil {
		existingKeys := utils.KeysOfSyncMap(r.sdkClients)
		remoteKeys := utils.KeysOfMap(autoConfig.SDKs)

		toAddKeys := utils.Except(remoteKeys, existingKeys)
		toDeleteKeys := utils.Except(existingKeys, remoteKeys)
		remainingKeys := utils.Except(remoteKeys, toAddKeys)

		r.deleteSdkClients(toDeleteKeys)
		if r.shouldUpdateOptions(&autoConfig.Options) { // Global options changed, reset all remaining SDKs
			r.options = &autoConfig.Options
			r.resetSdkClients(remainingKeys, autoConfig)
		} else { // Check SDK keys and update them if needed
			var resetKeys []string
			for _, sdkId := range remainingKeys {
				if sdkClient, ok := r.sdkClients.Load(sdkId); ok {
					if sdkConfig, ok := autoConfig.SDKs[sdkId]; ok {
						key1, key2 := sdkClient.SdkKeys()
						if key1 != sdkConfig.Key1 { // Primary SDK key changed, reset client
							resetKeys = append(resetKeys, sdkId)
							continue
						}
						if key2 == nil && sdkConfig.Key2 != "" { // Secondary SDK key added, update client
							sdkClient.SetSecondarySdkKey(sdkConfig.Key2)
							r.sdkClientsBySdkKey.Store(sdkConfig.Key2, sdkClient)
						}
						if key2 != nil && *key2 != sdkConfig.Key2 { // Secondary SDK key changed, update client
							sdkClient.SetSecondarySdkKey(sdkConfig.Key2)
							r.sdkClientsBySdkKey.Delete(*key2)
							r.sdkClientsBySdkKey.Store(sdkConfig.Key2, sdkClient)
						}
					}
				}
			}
			if len(resetKeys) > 0 {
				r.resetSdkClients(resetKeys, autoConfig)
			}
		}
		r.addSdkClients(toAddKeys, autoConfig)
	}
}

func (r *autoRegistrar) getConfig(ctx context.Context) (*model.ProxyConfigModel, error) {
	ctx, span := r.telemetryReporter.StartSpan(ctx, "profile poll")
	defer span.End()

	cachedConfig, cachedEtag, cacheErr := r.readCache(ctx)
	if r.conf.GlobalOfflineConfig.Enabled {
		if cacheErr != nil {
			return nil, fmt.Errorf("could not load proxy profile from cache: %s", cacheErr)
		}
		return r.parseConfig(cachedConfig, cachedEtag)
	}
	fetched, fetchedEtag, fetchErr := r.fetchConfig(ctx, cachedEtag)
	if fetchErr != nil {
		r.log.Errorf("could not fetch proxy profile, falling back to cache: %v", fetchErr)
		if cacheErr != nil {
			return nil, fmt.Errorf("could not load proxy profile from cache: %s", cacheErr)
		}
		return r.parseConfig(cachedConfig, cachedEtag)
	}
	if fetched == nil { // 304
		r.log.Debugf("proxy profile not modified")
		return r.parseConfig(cachedConfig, cachedEtag)
	} else { // 200
		r.log.Debugf("proxy profile fetched with etag %s", fetchedEtag)
		err := r.writeCache(ctx, fetched, fetchedEtag)
		if err != nil {
			r.log.Errorf("could not write proxy profile to cache: %v", err)
		}
		return r.parseConfig(fetched, fetchedEtag)
	}
}

func (r *autoRegistrar) parseConfig(body []byte, etag string) (*model.ProxyConfigModel, error) {
	if etag == r.etag {
		return nil, nil
	}
	parsed := model.ProxyConfigModel{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("error during parsing proxy profile: %v", err)
	}
	r.etag = etag
	r.log.Debugf("proxy profile loaded, got %d SDK keys", len(parsed.SDKs))
	return &parsed, nil
}

func (r *autoRegistrar) fetchConfig(ctx context.Context, etag string) ([]byte, string, error) {
	apiUrl := r.conf.Profile.BaseUrl + "/v1/proxy/config/" + r.conf.Profile.Key
	r.log.Debugf("fetching proxy profile from %s with etag %s", apiUrl, etag)
	request, err := http.NewRequestWithContext(ctx, "GET", apiUrl, nil)
	if err != nil {
		return nil, "", fmt.Errorf("error during fetching proxy profile: %v", err)
	}
	if etag != "" {
		request.Header.Set("If-None-Match", etag)
	}
	request.Header.Set("Authorization", "Bearer "+r.conf.Profile.Secret)
	response, err := r.httpClient.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("error during fetching proxy profile: %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode == http.StatusNotModified {
		return nil, "", nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("error during fetching proxy profile, status code: %d", response.StatusCode)
	}
	receivedEtag := response.Header.Get("ETag")
	if strings.HasPrefix(receivedEtag, "W/") {
		receivedEtag = receivedEtag[2:]
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", fmt.Errorf("error during reading proxy profile response: %v", err)
	}
	return body, receivedEtag, nil
}

func (r *autoRegistrar) buildSdkConfig(sdkId string, sdkModel *model.SdkConfigModel) *config.SDKConfig {
	sdkConfig := &config.SDKConfig{
		BaseUrl:                  r.conf.Profile.SDKs.BaseUrl,
		Key:                      sdkModel.Key1,
		PollInterval:             r.options.PollInterval,
		DataGovernance:           r.options.DataGovernance,
		Log:                      r.conf.Profile.SDKs.Log,
		WebhookSigningKey:        r.conf.Profile.WebhookSigningKey,
		WebhookSignatureValidFor: r.conf.Profile.WebhookSignatureValidFor,
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

func (r *autoRegistrar) buildSdkClient(sdkId string, sdkConfig *config.SDKConfig, sdkModel *model.SdkConfigModel) Client {
	ctx := &Context{
		SDKConf:            sdkConfig,
		TelemetryReporter:  r.telemetryReporter,
		StatusReporter:     r.statusReporter,
		GlobalDefaultAttrs: r.conf.DefaultAttrs,
		SdkId:              sdkId,
		ExternalCache:      r.cache,
		Transport:          r.sdkTransport,
	}
	if len(sdkModel.Key2) > 0 {
		ctx.SecondarySdkKey.Store(&sdkModel.Key2)
	}
	return NewClient(ctx, r.log)
}

func (r *autoRegistrar) deleteSdkClients(sdkIds []string) {
	if len(sdkIds) == 0 {
		r.log.Debugf("no SDK clients to remove")
		return
	}

	r.log.Debugf("removing %d SDK clients", len(sdkIds))
	for _, sdkId := range sdkIds {
		if sdkClient, ok := r.sdkClients.LoadAndDelete(sdkId); ok {
			key1, key2 := sdkClient.SdkKeys()
			if key1 != "" {
				r.sdkClientsBySdkKey.Delete(key1)
			}
			if key2 != nil && *key2 != "" {
				r.sdkClientsBySdkKey.Delete(*key2)
			}
			sdkClient.Close()
			r.statusReporter.RemoveSdk(sdkId)
			r.Publish(sdkId)
		}
	}
}

func (r *autoRegistrar) addSdkClients(sdkIds []string, config *model.ProxyConfigModel) {
	if len(sdkIds) == 0 {
		r.log.Debugf("no SDK clients to add")
		return
	}

	r.log.Debugf("adding %d SDK clients", len(sdkIds))
	for _, sdkId := range sdkIds {
		if sdkModel, ok := config.SDKs[sdkId]; ok {
			sdkConfig := r.buildSdkConfig(sdkId, sdkModel)
			if cl, loaded := r.sdkClients.LoadOrCompute(sdkId, func() Client {
				return r.buildSdkClient(sdkId, sdkConfig, sdkModel)
			}); !loaded {
				if sdkModel.Key1 != "" {
					r.sdkClientsBySdkKey.Store(sdkModel.Key1, cl)
				}
				if sdkModel.Key2 != "" {
					r.sdkClientsBySdkKey.Store(sdkModel.Key2, cl)
				}
				r.statusReporter.RegisterSdk(sdkId, sdkConfig)
				r.Publish(sdkId)
			}
		}
	}
}

func (r *autoRegistrar) resetSdkClients(sdkIds []string, config *model.ProxyConfigModel) {
	if len(sdkIds) == 0 {
		r.log.Debugf("no SDK clients to reset")
		return
	}

	r.log.Debugf("resetting %d SDK clients", len(sdkIds))
	for _, sdkId := range sdkIds {
		if sdkModel, ok := config.SDKs[sdkId]; ok {
			sdkConfig := r.buildSdkConfig(sdkId, sdkModel)
			sdkClient := r.buildSdkClient(sdkId, sdkConfig, sdkModel)
			if existing, loaded := r.sdkClients.LoadAndStore(sdkId, sdkClient); loaded {
				key1, key2 := existing.SdkKeys()
				if key1 != "" {
					r.sdkClientsBySdkKey.Delete(key1)
				}
				if key2 != nil && *key2 != "" {
					r.sdkClientsBySdkKey.Delete(*key2)
				}
				existing.Close()
				r.statusReporter.UpdateSdk(sdkId, sdkConfig)
				r.Publish(sdkId)
			}
			if sdkModel.Key1 != "" {
				r.sdkClientsBySdkKey.Store(sdkModel.Key1, sdkClient)
			}
			if sdkModel.Key2 != "" {
				r.sdkClientsBySdkKey.Store(sdkModel.Key2, sdkClient)
			}
		}
	}
}

func (r *autoRegistrar) shouldUpdateOptions(options *model.OptionsModel) bool {
	return r.options.PollInterval != options.PollInterval ||
		r.options.DataGovernance != options.DataGovernance
}

func (r *autoRegistrar) readCache(ctx context.Context) (config []byte, eTag string, err error) {
	if r.cache == nil {
		return nil, r.etag, fmt.Errorf("cache is not configured")
	}
	cached, err := r.cache.Get(ctx, r.cacheKey)
	if err != nil {
		return nil, r.etag, err
	}
	if len(cached) == 0 {
		return nil, "", fmt.Errorf("no valid proxy profile found in cache")
	}
	return cacheSegmentsFromBytes(cached)
}

func (r *autoRegistrar) writeCache(ctx context.Context, config []byte, etag string) error {
	if r.cache == nil {
		return fmt.Errorf("cache is not configured")
	}
	err := r.cache.Set(ctx, r.cacheKey, cacheSegmentsToBytes(etag, config))
	if err != nil {
		return err
	}
	return nil
}

const newLineByte byte = '\n'

func cacheSegmentsFromBytes(cacheBytes []byte) (config []byte, eTag string, err error) {
	eTagIndex := bytes.IndexByte(cacheBytes, newLineByte)
	if eTagIndex == -1 {
		return nil, "", fmt.Errorf("number of values is fewer than expected")
	}

	eTagBytes := cacheBytes[:eTagIndex]
	if len(eTagBytes) == 0 {
		return nil, "", fmt.Errorf("empty eTag value")
	}

	configBytes := cacheBytes[eTagIndex+1:]
	if len(configBytes) == 0 {
		return nil, "", fmt.Errorf("empty configuration JSON")
	}

	return configBytes, string(eTagBytes), nil
}

func cacheSegmentsToBytes(eTag string, config []byte) []byte {
	toCache := []byte(eTag)
	toCache = append(toCache, newLineByte)
	toCache = append(toCache, config...)
	return toCache
}
