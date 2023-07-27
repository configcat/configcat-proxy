package cache

import (
	"context"
	"encoding/json"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v8/configcatcache"
	"time"
)

type notifyingCacheStorage struct {
	store.CacheStorage
	store.Notifier

	poller    *time.Ticker
	log       log.Logger
	reporter  status.Reporter
	ctx       context.Context
	ctxCancel func()
	sdkId     string
	cacheKey  string
}

func NewNotifyingCacheStorage(sdkId string, sdkKey string, cache store.CacheStorage, conf *config.OfflineConfig, reporter status.Reporter, log log.Logger) store.NotifyingStorage {
	nrLogger := log.WithPrefix("cache-poll")
	n := &notifyingCacheStorage{
		CacheStorage: cache,
		Notifier:     store.NewNotifier(),
		cacheKey:     configcatcache.ProduceCacheKey(sdkKey),
		reporter:     reporter,
		log:          nrLogger,
		sdkId:        sdkId,
		poller:       time.NewTicker(time.Duration(conf.CachePollInterval) * time.Second),
	}
	n.ctx, n.ctxCancel = context.WithCancel(context.Background())
	n.reload()
	go n.run()
	return n
}

func (n *notifyingCacheStorage) run() {
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

func (n *notifyingCacheStorage) reload() bool {
	data, err := n.CacheStorage.Get(n.ctx, n.cacheKey)
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
	if n.LoadEntry().CachedETag == eTag {
		n.reporter.ReportOk(n.sdkId, "config from cache not modified")
		return false
	}
	n.log.Debugf("new JSON received from redis, reloading")

	var root store.RootNode
	if err = json.Unmarshal(configJson, &root); err != nil {
		n.log.Errorf("failed to parse JSON from redis: %s", err)
		n.reporter.ReportError(n.sdkId, err)
		return false
	}
	root.Fixup()
	ser, _ := json.Marshal(root) // Re-serialize to enforce the JSON schema
	n.StoreEntry(ser, fetchTime, eTag)
	n.reporter.ReportOk(n.sdkId, "reload from cache succeeded")
	return true
}

func (n *notifyingCacheStorage) Get(_ context.Context, _ string) ([]byte, error) {
	return n.ComposeBytes(), nil
}

func (n *notifyingCacheStorage) Set(_ context.Context, _ string, _ []byte) error {
	return nil // do nothing
}

func (n *notifyingCacheStorage) Close() {
	n.Notifier.Close()
	n.ctxCancel()
	n.poller.Stop()
	n.CacheStorage.Close()
	n.log.Reportf("shutdown complete")
}
