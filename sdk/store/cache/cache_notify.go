package cache

import (
	"context"
	"encoding/json"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/configcat-proxy/status"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"time"
)

type notifyingCacheStore struct {
	store.CacheEntryStore
	store.Notifier

	poller    *time.Ticker
	log       log.Logger
	reporter  status.Reporter
	ctx       context.Context
	ctxCancel func()
	sdkId     string
	cacheKey  string
}

var _ store.NotifyingStore = &notifyingCacheStore{}

func NewNotifyingCacheStore(sdkId string, cacheKey string, cache store.CacheEntryStore, conf *config.OfflineConfig, reporter status.Reporter, log log.Logger) configcat.ConfigCache {
	nrLogger := log.WithPrefix("cache-poll")
	n := &notifyingCacheStore{
		CacheEntryStore: cache,
		Notifier:        store.NewNotifier(),
		cacheKey:        cacheKey,
		reporter:        reporter,
		log:             nrLogger,
		sdkId:           sdkId,
		poller:          time.NewTicker(time.Duration(conf.CachePollInterval) * time.Second),
	}
	n.ctx, n.ctxCancel = context.WithCancel(context.Background())
	n.reload()
	go n.run()
	return n
}

func (n *notifyingCacheStore) run() {
	for {
		select {
		case <-n.poller.C:
			if n.reload() {
				n.Notify()
			}
		case <-n.Closed():
			return
		}
	}
}

func (n *notifyingCacheStore) reload() bool {
	data, err := n.CacheEntryStore.Get(n.ctx, n.cacheKey)
	if err != nil {
		n.log.Errorf("failed to read from redis: %s", err)
		n.reporter.ReportError(n.sdkId, err)
		return false
	}
	fetchTime, eTag, configJson, err := configcatcache.CacheSegmentsFromBytes(data)
	if err != nil {
		n.log.Errorf("failed to recognise the cache format: %s", err)
		n.reporter.ReportError(n.sdkId, err)
		return false
	}
	if n.LoadEntry().ETag == eTag {
		n.reporter.ReportOk(n.sdkId, "config from cache not modified")
		return false
	}
	n.log.Debugf("new JSON received from redis, reloading")

	var root configcat.ConfigJson
	if err = json.Unmarshal(configJson, &root); err != nil {
		n.log.Errorf("failed to parse JSON from redis: %s", err)
		n.reporter.ReportError(n.sdkId, err)
		return false
	}
	ser, _ := json.Marshal(root) // Re-serialize to enforce the JSON schema
	n.CacheEntryStore.StoreEntry(ser, fetchTime, eTag)
	n.reporter.ReportOk(n.sdkId, "reload from cache succeeded")
	return true
}

func (n *notifyingCacheStore) Get(_ context.Context, _ string) ([]byte, error) {
	return n.CacheEntryStore.ComposeBytes(), nil
}

func (n *notifyingCacheStore) Set(_ context.Context, _ string, _ []byte) error {
	return nil // do nothing
}

func (n *notifyingCacheStore) Close() {
	n.Notifier.Close()
	n.ctxCancel()
	n.poller.Stop()
	if closable, ok := n.CacheEntryStore.(store.ClosableStore); ok {
		closable.Close()
	}
	n.log.Reportf("shutdown complete")
}
