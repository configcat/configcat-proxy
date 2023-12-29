package sdk

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/sdk/statistics"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/configcat-proxy/sdk/store/cache"
	"github.com/configcat/configcat-proxy/sdk/store/cache/redis"
	"github.com/configcat/configcat-proxy/sdk/store/file"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

const (
	validEmptySdkKey = "0000000000000000000000/0000000000000000000000"
)

type Client interface {
	Eval(key string, user UserAttrs) (EvalData, error)
	EvalAll(user UserAttrs) map[string]EvalData
	Keys() []string
	GetCachedJson() *store.EntryWithEtag
	SubConfigChanged(id string) <-chan struct{}
	UnsubConfigChanged(id string)
	Ready() <-chan struct{}
	Refresh() error
	Close()
	WebhookSigningKey() string
	WebhookSignatureValidFor() int
}

type EvalData struct {
	Value       interface{}
	VariationId string
	User        configcat.User
}

type Context struct {
	SdkId              string
	SDKConf            *config.SDKConfig
	ProxyConf          *config.HttpProxyConfig
	CacheConf          *config.CacheConfig
	GlobalDefaultAttrs UserAttrs
	MetricsHandler     metrics.Handler
	StatusReporter     status.Reporter
	EvalReporter       statistics.Reporter
}

type client struct {
	configCatClient *configcat.Client
	defaultAttrs    UserAttrs
	subscriptions   map[string]chan struct{}
	readyOnce       sync.Once
	log             log.Logger
	cache           store.EntryStore
	sdkCtx          *Context
	mu              sync.RWMutex
	initialized     atomic.Bool
	ctx             context.Context
	ctxCancel       func()
}

func NewClient(sdkCtx *Context, log log.Logger) Client {
	sdkLog := log.WithLevel(sdkCtx.SDKConf.Log.GetLevel()).WithPrefix("sdk-" + sdkCtx.SdkId)
	offline := sdkCtx.SDKConf.Offline.Enabled
	key := sdkCtx.SDKConf.Key
	var storage configcat.ConfigCache
	if offline && sdkCtx.SDKConf.Offline.Local.FilePath != "" {
		key = validEmptySdkKey
		storage = file.NewFileStore(sdkCtx.SdkId, &sdkCtx.SDKConf.Offline.Local, sdkCtx.StatusReporter, log.WithLevel(sdkCtx.SDKConf.Offline.Log.GetLevel()))
	} else if offline && sdkCtx.SDKConf.Offline.UseCache && sdkCtx.CacheConf.Redis.Enabled {
		cacheKey := configcatcache.ProduceCacheKey(key, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
		cacheStore := cache.NewCacheStore(redis.NewRedisStore(&sdkCtx.CacheConf.Redis), sdkCtx.StatusReporter)
		storage = cache.NewNotifyingCacheStore(sdkCtx.SdkId, cacheKey, cacheStore, &sdkCtx.SDKConf.Offline, sdkCtx.StatusReporter, log.WithLevel(sdkCtx.SDKConf.Offline.Log.GetLevel()))
	} else if !offline && sdkCtx.CacheConf.Redis.Enabled {
		storage = cache.NewCacheStore(redis.NewRedisStore(&sdkCtx.CacheConf.Redis), sdkCtx.StatusReporter)
	} else {
		storage = store.NewInMemoryStorage()
	}
	client := &client{
		log:           sdkLog,
		subscriptions: make(map[string]chan struct{}),
		cache:         storage.(store.EntryStore),
		sdkCtx:        sdkCtx,
		defaultAttrs:  MergeUserAttrs(sdkCtx.GlobalDefaultAttrs, sdkCtx.SDKConf.DefaultAttrs),
	}
	client.ctx, client.ctxCancel = context.WithCancel(context.Background())
	var transport = http.DefaultTransport.(*http.Transport)
	if !sdkCtx.SDKConf.Offline.Enabled && sdkCtx.ProxyConf.Url != "" {
		proxyUrl, err := url.Parse(sdkCtx.ProxyConf.Url)
		if err != nil {
			sdkLog.Errorf("failed to parse proxy url: %s", sdkCtx.ProxyConf.Url)
		} else {
			transport.Proxy = http.ProxyURL(proxyUrl)
			sdkLog.Reportf("using HTTP proxy: %s", sdkCtx.ProxyConf.Url)
		}
	}
	clientConfig := configcat.Config{
		PollingMode:    configcat.AutoPoll,
		PollInterval:   time.Duration(sdkCtx.SDKConf.PollInterval) * time.Second,
		Offline:        offline,
		BaseURL:        sdkCtx.SDKConf.BaseUrl,
		Cache:          storage,
		SDKKey:         key,
		DataGovernance: configcat.Global,
		Logger:         sdkLog,
		LogLevel:       sdkLog.GetLevel(),
		Transport:      OverrideUserAgent(transport),
		Hooks:          &configcat.Hooks{},
	}
	if !sdkCtx.SDKConf.Offline.Enabled {
		clientConfig.Hooks.OnConfigChanged = func() {
			client.signal()
		}
		if sdkCtx.MetricsHandler != nil {
			clientConfig.Transport = metrics.InterceptSdk(sdkCtx.SdkId, sdkCtx.MetricsHandler, clientConfig.Transport)
		}
		clientConfig.Transport = status.InterceptSdk(sdkCtx.SdkId, sdkCtx.StatusReporter, clientConfig.Transport)
	} else {
		clientConfig.PollingMode = configcat.Manual
	}
	if sdkCtx.EvalReporter != nil {
		clientConfig.Hooks.OnFlagEvaluated = func(details *configcat.EvaluationDetails) {
			var user map[string]string
			if details.Data.User != nil {
				if userAttrs, ok := details.Data.User.(UserAttrs); ok && userAttrs != nil {
					user = userAttrs
				}
			}
			sdkCtx.EvalReporter.ReportEvaluation(&statistics.EvalEvent{
				SdkId:     sdkCtx.SdkId,
				FlagKey:   details.Data.Key,
				Value:     details.Value,
				UserAttrs: user})
		}
	}
	if sdkCtx.SDKConf.DataGovernance == "eu" {
		clientConfig.DataGovernance = configcat.EUOnly
	}
	client.configCatClient = configcat.NewCustomClient(clientConfig)

	// sync up values to the SDK from the OFFLINE store
	if sdkCtx.SDKConf.Offline.Enabled {
		_ = client.Refresh()
	}

	if notifier, ok := storage.(store.NotifyingStore); ok {
		go client.listen(notifier)
	}
	return client
}

func (c *client) listen(notifier store.NotifyingStore) {
	for {
		select {
		case <-notifier.Modified():
			c.signal()
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *client) signal() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// we don't want to notify subscribers in ONLINE mode
	// about the first change upon SDK initialization
	if !c.sdkCtx.SDKConf.Offline.Enabled && c.initialized.CompareAndSwap(false, true) {
		return
	}
	// force the SDK to reload local values in OFFLINE mode
	if c.sdkCtx.SDKConf.Offline.Enabled {
		_ = c.Refresh()
	}
	for _, sub := range c.subscriptions {
		sub <- struct{}{}
	}
}

func (c *client) Eval(key string, user UserAttrs) (EvalData, error) {
	mergedUser := MergeUserAttrs(c.defaultAttrs, user)
	details := c.configCatClient.Snapshot(mergedUser).GetValueDetails(key)
	return EvalData{Value: details.Value, VariationId: details.Data.VariationID, User: details.Data.User}, details.Data.Error
}

func (c *client) EvalAll(user UserAttrs) map[string]EvalData {
	mergedUser := MergeUserAttrs(c.defaultAttrs, user)
	allDetails := c.configCatClient.Snapshot(mergedUser).GetAllValueDetails()
	result := make(map[string]EvalData, len(allDetails))
	for _, details := range allDetails {
		result[details.Data.Key] = EvalData{Value: details.Value, VariationId: details.Data.VariationID, User: details.Data.User}
	}
	return result
}

func (c *client) Keys() []string {
	return c.configCatClient.GetAllKeys()
}

func (c *client) GetCachedJson() *store.EntryWithEtag {
	c.readyOnce.Do(func() {
		select {
		case <-c.configCatClient.Ready():
		case <-c.ctx.Done():
		}
	})
	return c.cache.LoadEntry()
}

func (c *client) SubConfigChanged(id string) <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	sub, ok := c.subscriptions[id]
	if !ok {
		sub = make(chan struct{}, 1)
		c.subscriptions[id] = sub
	}
	return sub
}

func (c *client) UnsubConfigChanged(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.subscriptions, id)
}

func (c *client) Refresh() error {
	return c.configCatClient.Refresh(c.ctx)
}

func (c *client) Ready() <-chan struct{} {
	return c.configCatClient.Ready()
}

func (c *client) WebhookSigningKey() string {
	return c.sdkCtx.SDKConf.WebhookSigningKey
}

func (c *client) WebhookSignatureValidFor() int {
	return c.sdkCtx.SDKConf.WebhookSignatureValidFor
}

func (c *client) Close() {
	c.ctxCancel()
	if closable, ok := c.cache.(store.ClosableStore); ok {
		closable.Close()
	}
	c.configCatClient.Close()
	c.log.Reportf("shutdown complete")
}
