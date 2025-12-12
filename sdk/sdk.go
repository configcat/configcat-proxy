package sdk

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/configcat/configcat-proxy/cache"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/pubsub"
	"github.com/configcat/configcat-proxy/sdk/statistics"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/configcat-proxy/sdk/store/file"
	"github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
)

const (
	validEmptySdkKey = "0000000000000000000000/0000000000000000000000"
)

type Client interface {
	pubsub.SubscriptionHandler[struct{}]
	Eval(key string, user model.UserAttrs) model.EvalData
	EvalAll(user model.UserAttrs) map[string]model.EvalData
	Keys() []string
	HasKey(key string) bool
	GetCachedJson() *store.EntryWithEtag
	Refresh(ctx context.Context) error
	Close()
	SdkKeys() (string, *string)
	SetSecondarySdkKey(sdkKey string)
	WebhookSigningKey() string
	WebhookSignatureValidFor() int
	IsInValidState() bool
	Ready() <-chan struct{}
}

type Context struct {
	SdkId              string
	SecondarySdkKey    atomic.Pointer[string]
	SDKConf            *config.SDKConfig
	GlobalDefaultAttrs model.UserAttrs
	TelemetryReporter  telemetry.Reporter
	StatusReporter     status.Reporter
	EvalReporter       statistics.Reporter
	ExternalCache      cache.ReaderWriter
	Transport          http.RoundTripper
}

type client struct {
	configCatClient *configcat.Client
	defaultAttrs    model.UserAttrs
	log             log.Logger
	cache           store.EntryStore
	sdkCtx          *Context
	initialized     atomic.Bool
	ready           chan struct{}
	readyOnce       sync.Once
	ctx             context.Context
	ctxCancel       func()
	mu              sync.Mutex
	pubsub.Publisher[struct{}]
}

func NewClient(sdkCtx *Context, log log.Logger) Client {
	sdkLog := log.WithLevel(sdkCtx.SDKConf.Log.GetLevel()).WithPrefix(sdkCtx.SdkId)

	offline := sdkCtx.SDKConf.Offline.Enabled
	key := sdkCtx.SDKConf.Key
	var storage configcat.ConfigCache
	if offline && sdkCtx.SDKConf.Offline.Local.FilePath != "" {
		key = validEmptySdkKey
		storage = file.NewFileStore(sdkCtx.SdkId, &sdkCtx.SDKConf.Offline.Local, sdkCtx.StatusReporter, log.WithLevel(sdkCtx.SDKConf.Offline.Log.GetLevel()))
	} else if offline && sdkCtx.SDKConf.Offline.UseCache && sdkCtx.ExternalCache != nil {
		cacheKey := configcatcache.ProduceCacheKey(sdkCtx.SDKConf.Key, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
		cacheStore := store.NewCacheStore(sdkCtx.ExternalCache, sdkCtx.StatusReporter)
		storage = store.NewNotifyingCacheStore(sdkCtx.SdkId, cacheKey, cacheStore, &sdkCtx.SDKConf.Offline, sdkCtx.TelemetryReporter, sdkCtx.StatusReporter, log.WithLevel(sdkCtx.SDKConf.Offline.Log.GetLevel()))
	} else if !offline && sdkCtx.ExternalCache != nil {
		storage = store.NewCacheStore(sdkCtx.ExternalCache, sdkCtx.StatusReporter)
	} else {
		storage = store.NewInMemoryStorage()
	}
	client := &client{
		Publisher:    pubsub.NewPublisher[struct{}](),
		log:          sdkLog,
		cache:        storage.(store.EntryStore),
		sdkCtx:       sdkCtx,
		ready:        make(chan struct{}),
		defaultAttrs: model.MergeUserAttrs(sdkCtx.GlobalDefaultAttrs, sdkCtx.SDKConf.DefaultAttrs),
	}
	client.ctx, client.ctxCancel = context.WithCancel(context.Background())
	clientConfig := configcat.Config{
		PollingMode:    configcat.Manual,
		Offline:        offline,
		BaseURL:        sdkCtx.SDKConf.BaseUrl,
		Cache:          storage,
		SDKKey:         key,
		DataGovernance: configcat.Global,
		Logger:         sdkLog,
		LogLevel:       sdkLog.GetConfigCatLevel(),
		Transport:      sdkCtx.Transport,
		Hooks:          &configcat.Hooks{},
	}
	if !offline {
		clientConfig.Hooks.OnConfigChanged = func() {
			client.signal()
		}
		clientConfig.Transport = sdkCtx.TelemetryReporter.InstrumentHttpClient(
			status.InterceptSdk(sdkCtx.SdkId, sdkCtx.StatusReporter, clientConfig.Transport),
			telemetry.SdkId.V(sdkCtx.SdkId),
			telemetry.Source.V("sdk"))

	}
	if sdkCtx.EvalReporter != nil {
		clientConfig.Hooks.OnFlagEvaluated = func(details *configcat.EvaluationDetails) {
			var user map[string]interface{}
			if details.Data.User != nil {
				if userAttrs, ok := details.Data.User.(model.UserAttrs); ok && userAttrs != nil {
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
	go func() {
		client.mu.Lock()
		defer client.mu.Unlock()
		_ = client.Refresh(client.ctx)
		close(client.ready)
	}()

	if notifier, ok := storage.(store.NotifyingStore); ok {
		go client.listen(notifier)
	} else {
		go client.poll()
	}
	sdkLog.Reportf("started")
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

func (c *client) poll() {
	interval := c.sdkCtx.SDKConf.PollInterval
	if interval < 1 {
		interval = config.DefaultSdkPollInterval
	}
	poller := time.NewTicker(time.Duration(interval) * time.Second)
	defer poller.Stop()
	for {
		select {
		case <-poller.C:
			spanCtx, span := c.sdkCtx.TelemetryReporter.StartSpan(c.ctx, c.sdkCtx.SdkId+" poll")
			_ = c.Refresh(spanCtx)
			span.End()
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *client) signal() {
	// we don't want to notify subscribers in ONLINE mode
	// about the first change upon SDK initialization
	if !c.sdkCtx.SDKConf.Offline.Enabled && c.initialized.CompareAndSwap(false, true) {
		return
	}
	// force the SDK to reload local values in OFFLINE mode
	if c.sdkCtx.SDKConf.Offline.Enabled {
		_ = c.Refresh(c.ctx)
	}
	c.Publish(struct{}{})
}

func (c *client) ensureReady() {
	c.readyOnce.Do(func() {
		select {
		case <-c.ready:
		case <-c.ctx.Done():
		}
	})
}

func (c *client) Eval(key string, user model.UserAttrs) model.EvalData {
	c.ensureReady()
	mergedUser := model.MergeUserAttrs(c.defaultAttrs, user)
	details := c.configCatClient.Snapshot(mergedUser).GetValueDetails(key)
	return model.EvalData{Value: details.Value, VariationId: details.Data.VariationID, User: details.Data.User, Error: details.Data.Error,
		IsTargeting: details.Data.MatchedPercentageOption != nil || details.Data.MatchedTargetingRule != nil}
}

func (c *client) EvalAll(user model.UserAttrs) map[string]model.EvalData {
	c.ensureReady()
	mergedUser := model.MergeUserAttrs(c.defaultAttrs, user)
	allDetails := c.configCatClient.Snapshot(mergedUser).GetAllValueDetails()
	result := make(map[string]model.EvalData, len(allDetails))
	for _, details := range allDetails {
		result[details.Data.Key] = model.EvalData{Value: details.Value, VariationId: details.Data.VariationID, User: details.Data.User, Error: details.Data.Error,
			IsTargeting: details.Data.MatchedPercentageOption != nil || details.Data.MatchedTargetingRule != nil}
	}
	return result
}

func (c *client) Keys() []string {
	c.ensureReady()
	return c.configCatClient.GetAllKeys()
}

func (c *client) HasKey(key string) bool {
	c.ensureReady()
	keys := c.configCatClient.GetAllKeys()
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

func (c *client) GetCachedJson() *store.EntryWithEtag {
	c.ensureReady()
	return c.cache.LoadEntry()
}

func (c *client) Refresh(ctx context.Context) error {
	spanCtx, span := c.sdkCtx.TelemetryReporter.StartSpan(ctx, c.sdkCtx.SdkId+" refresh")
	defer span.End()

	return c.configCatClient.RefreshWithContext(spanCtx)
}

func (c *client) SdkKeys() (string, *string) {
	secondary := c.sdkCtx.SecondarySdkKey.Load()
	return c.sdkCtx.SDKConf.Key, secondary
}

func (c *client) SetSecondarySdkKey(sdkKey string) {
	c.log.Debugf("setting secondary SDK key")
	c.sdkCtx.SecondarySdkKey.Store(&sdkKey)
}

func (c *client) WebhookSigningKey() string {
	return c.sdkCtx.SDKConf.WebhookSigningKey
}

func (c *client) WebhookSignatureValidFor() int {
	return c.sdkCtx.SDKConf.WebhookSignatureValidFor
}

func (c *client) IsInValidState() bool {
	return !c.GetCachedJson().Empty
}

func (c *client) Ready() <-chan struct{} {
	return c.ready
}

func (c *client) Close() {
	if notifier, ok := c.cache.(store.Notifier); ok {
		notifier.Close()
	}
	c.Publisher.Close()
	c.ctxCancel()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configCatClient.Close()
	c.log.Reportf("shutdown complete")
}

func Version() string {
	return proxyVersion
}
