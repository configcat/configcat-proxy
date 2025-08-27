package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
)

type notifyingCacheStore struct {
	store.CacheEntryStore
	store.Notifier

	log      log.Logger
	reporter status.Reporter
	sdkId    string
	cacheKey string
}

func NewNotifyingCacheStore(sdkId string, cacheKey string, cache store.CacheEntryStore, conf *config.OfflineConfig, reporter status.Reporter, log log.Logger) store.NotifyingStore {
	nrLogger := log.WithPrefix("cache-poll")
	n := &notifyingCacheStore{
		CacheEntryStore: cache,
		Notifier:        store.NewNotifier(),
		cacheKey:        cacheKey,
		reporter:        reporter,
		log:             nrLogger,
		sdkId:           sdkId,
	}
	n.reload()
	go n.run(conf.CachePollInterval)
	return n
}

func (n *notifyingCacheStore) run(interval int) {
	poller := time.NewTicker(time.Duration(interval) * time.Second)
	defer poller.Stop()
	for {
		select {
		case <-poller.C:
			if n.reload() {
				n.Notify()
			}
		case <-n.Notifier.Context().Done():
			return
		}
	}
}

func (n *notifyingCacheStore) reload() bool {
	data, err := n.CacheEntryStore.Get(n.Notifier.Context(), n.cacheKey)
	if err != nil {
		n.log.Errorf("failed to read from cache: %s", err)
		n.reporter.ReportError(n.sdkId, "failed to read from cache")
		return false
	}
	fetchTime, eTag, configJson, err := configcatcache.CacheSegmentsFromBytes(data)
	if err != nil {
		n.log.Errorf("failed to recognise the cache format: %s", err)
		n.reporter.ReportError(n.sdkId, "failed to recognise the cache format")
		return false
	}
	if n.LoadEntry().ETag == eTag {
		n.reporter.ReportOk(n.sdkId, "config from cache not modified")
		return false
	}
	n.log.Debugf("new JSON received from cache, reloading")

	var root configcat.ConfigJson
	if err = json.Unmarshal(configJson, &root); err != nil {
		n.log.Errorf("failed to parse JSON from cache: %s", err)
		n.reporter.ReportError(n.sdkId, "failed to parse JSON from cache")
		return false
	}
	n.CacheEntryStore.StoreEntry(configJson, fetchTime, eTag)
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
	n.log.Reportf("shutdown complete")
}
