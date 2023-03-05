package sdk

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/configcat-proxy/sdk/store/cache"
	"github.com/configcat/configcat-proxy/sdk/store/cache/redis"
	"github.com/configcat/configcat-proxy/sdk/store/file"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v7"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Client interface {
	Eval(key string, user *UserAttrs) (EvalData, error)
	EvalAll(user *UserAttrs) map[string]EvalData
	Keys() []string
	GetCachedJson() *store.EntryWithEtag
	SubConfigChanged(id string) <-chan struct{}
	Ready() <-chan struct{}
	UnsubConfigChanged(id string)
	Refresh() error
	Close()
}

type EvalData struct {
	Value       interface{}
	VariationId string
}

type client struct {
	configCatClient *configcat.Client
	subscriptions   map[string]chan struct{}
	stop            chan struct{}
	ready           chan struct{}
	readyOnce       sync.Once
	log             log.Logger
	cache           store.Storage
	conf            config.SDKConfig
	mu              sync.RWMutex
	initialized     atomic.Bool
	ctx             context.Context
	ctxCancel       func()
}

func NewClient(conf config.SDKConfig, metricsHandler metrics.Handler, reporter status.Reporter, log log.Logger) Client {
	sdkLog := log.WithLevel(conf.Log.GetLevel()).WithPrefix("sdk")
	var offline = conf.Offline.Enabled
	var storage store.Storage
	if offline && conf.Offline.Local.FilePath != "" {
		storage = file.NewFileStorage(conf.Offline.Local, reporter, log.WithLevel(conf.Offline.Log.GetLevel()))
	} else if offline && conf.Offline.UseCache && conf.Cache.Redis.Enabled {
		redisStore := redis.NewRedisStorage(conf.Key, conf.Cache.Redis, reporter)
		storage = cache.NewNotifyingCacheStorage(redisStore, conf.Offline, reporter, log.WithLevel(conf.Offline.Log.GetLevel()))
	} else if !offline && conf.Cache.Redis.Enabled {
		storage = redis.NewRedisStorage(conf.Key, conf.Cache.Redis, reporter)
	} else {
		storage = &store.InMemoryStorage{EntryStore: store.NewEntryStore()}
	}
	client := &client{
		log:           sdkLog,
		subscriptions: make(map[string]chan struct{}),
		stop:          make(chan struct{}),
		ready:         make(chan struct{}),
		cache:         storage,
		conf:          conf,
	}
	client.ctx, client.ctxCancel = context.WithCancel(context.Background())
	clientConfig := configcat.Config{
		PollingMode:    configcat.AutoPoll,
		PollInterval:   time.Duration(conf.PollInterval) * time.Second,
		Offline:        offline,
		BaseURL:        conf.BaseUrl,
		Cache:          storage,
		SDKKey:         conf.Key,
		DataGovernance: configcat.Global,
		Logger:         sdkLog,
		Transport:      http.DefaultTransport,
	}
	if !conf.Offline.Enabled {
		clientConfig.Hooks = &configcat.Hooks{
			OnConfigChanged: func() {
				client.signal()
			},
		}
		if metricsHandler != nil {
			clientConfig.Transport = metrics.InterceptSdk(metricsHandler, clientConfig.Transport)
		}
		clientConfig.Transport = status.InterceptSdk(reporter, clientConfig.Transport)
	} else {
		clientConfig.PollingMode = configcat.Manual
		close(client.ready) // in OFFLINE mode we are ready immediately
	}
	if conf.DataGovernance == "eu" {
		clientConfig.DataGovernance = configcat.EUOnly
	}
	client.configCatClient = configcat.NewCustomClient(clientConfig)

	// sync up values to the SDK from the OFFLINE store
	if conf.Offline.Enabled {
		_ = client.Refresh()
	}

	client.run()
	return client
}

func (c *client) run() {
	go func() {
		for {
			select {
			case <-c.cache.Modified():
				c.signal()
			case <-c.stop:
				return
			}
		}
	}()
}

func (c *client) signal() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// we don't want to notify subscribers in ONLINE mode
	// about the first change upon SDK initialization
	if !c.conf.Offline.Enabled && c.initialized.CompareAndSwap(false, true) {
		close(c.ready)
		return
	}
	// force the SDK to reload local values in OFFLINE mode
	if c.conf.Offline.Enabled {
		_ = c.Refresh()
	}
	for _, sub := range c.subscriptions {
		sub <- struct{}{}
	}
}

func (c *client) Eval(key string, user *UserAttrs) (EvalData, error) {
	details := c.configCatClient.Snapshot(user).GetValueDetails(key)
	return EvalData{Value: details.Value, VariationId: details.Data.VariationID}, details.Data.Error
}

func (c *client) EvalAll(user *UserAttrs) map[string]EvalData {
	allDetails := c.configCatClient.Snapshot(user).GetAllValueDetails()
	result := make(map[string]EvalData, len(allDetails))
	for _, details := range allDetails {
		result[details.Data.Key] = EvalData{Value: details.Value, VariationId: details.Data.VariationID}
	}
	return result
}

func (c *client) Keys() []string {
	return c.configCatClient.GetAllKeys()
}

func (c *client) GetCachedJson() *store.EntryWithEtag {
	c.readyOnce.Do(func() {
		select {
		case <-c.ready:
		case <-c.ctx.Done():
		}
	})
	return c.cache.GetLatestJson()
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
	return c.ready
}

func (c *client) Close() {
	close(c.stop)
	c.ctxCancel()
	c.cache.Close()
	c.configCatClient.Close()
	c.log.Reportf("shutdown complete")
}
