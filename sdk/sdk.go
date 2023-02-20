package sdk

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/configcat-proxy/sdk/store/file"
	"github.com/configcat/configcat-proxy/sdk/store/redis"
	"github.com/configcat/go-sdk/v7"
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
	closed          chan struct{}
	closedOnce      sync.Once
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

func NewClient(conf config.SDKConfig, log log.Logger) Client {
	sdkLog := log.WithLevel(conf.Log.GetLevel()).WithPrefix("sdk")
	var offline = conf.Offline.Enabled
	var cache store.Storage
	if offline && conf.Offline.Local.FilePath != "" {
		cache = file.NewFileStorage(conf.Offline.Local, log.WithLevel(conf.Offline.Log.GetLevel()))
	} else if offline && conf.Cache.Redis.Enabled {
		cache = redis.NewNotifyingRedisStorage(conf.Key, conf, log.WithLevel(conf.Offline.Log.GetLevel()))
	} else if !offline && conf.Cache.Redis.Enabled {
		cache = redis.NewRedisStorage(conf.Key, conf.Cache.Redis)
	} else {
		cache = &store.InMemoryStorage{EntryStore: store.NewEntryStore()}
	}
	client := &client{
		log:           sdkLog,
		subscriptions: make(map[string]chan struct{}),
		closed:        make(chan struct{}),
		ready:         make(chan struct{}),
		cache:         cache,
		conf:          conf,
	}
	client.ctx, client.ctxCancel = context.WithCancel(context.Background())
	clientConfig := configcat.Config{
		PollingMode:    configcat.AutoPoll,
		PollInterval:   time.Duration(conf.PollInterval) * time.Second,
		Offline:        offline,
		BaseURL:        conf.BaseUrl,
		Cache:          cache,
		SDKKey:         conf.Key,
		DataGovernance: configcat.Global,
		Logger:         sdkLog,
	}
	if !conf.Offline.Enabled {
		clientConfig.Hooks = &configcat.Hooks{
			OnConfigChanged: func() {
				client.signal()
			},
		}
	} else {
		clientConfig.PollingMode = configcat.Manual
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
	go func(cl *client) {
		for {
			select {
			case <-cl.cache.Modified():
				cl.signal()
			case <-cl.closed:
				c.ctxCancel()
				cl.cache.Close()
				cl.configCatClient.Close()
				cl.log.Reportf("shutdown complete")
				return
			}
		}
	}(c)
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
		if !c.conf.Offline.Enabled {
			select {
			case <-c.ready:
			case <-c.ctx.Done():
			}
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

func (c *client) Close() {
	c.closedOnce.Do(func() {
		close(c.closed)
	})
}