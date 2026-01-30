package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
)

type notifyingCacheStore struct {
	CacheEntryStore
	Notifier

	log               log.Logger
	statusReporter    status.Reporter
	telemetryReporter telemetry.Reporter
	sdkId             string
	cacheKey          string
}

func NewNotifyingCacheStore(sdkId string, cacheKey string, cache CacheEntryStore, conf *config.OfflineConfig,
	telemetryReporter telemetry.Reporter, statusReporter status.Reporter, log log.Logger) NotifyingStore {
	nrLogger := log.WithPrefix("cache-poll")
	n := &notifyingCacheStore{
		CacheEntryStore:   cache,
		Notifier:          NewNotifier(),
		cacheKey:          cacheKey,
		statusReporter:    statusReporter,
		telemetryReporter: telemetryReporter,
		log:               nrLogger,
		sdkId:             sdkId,
	}
	n.reload()
	go n.run(conf.CachePollInterval)
	return n
}

func (n *notifyingCacheStore) run(interval int) {
	inter := interval
	if inter < 1 {
		inter = config.DefaultCachePollInterval
	}
	poller := time.NewTicker(time.Duration(inter) * time.Second)
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
	ctx, span := n.telemetryReporter.StartSpan(n.Notifier.Context(), n.sdkId+" cache poll")
	defer span.End()

	data, err := n.CacheEntryStore.Get(ctx, n.cacheKey)
	if err != nil {
		n.log.Errorf("failed to read from cache: %s", err)
		n.statusReporter.ReportError(n.sdkId, "failed to read from cache")
		return false
	}
	fetchTime, eTag, configJson, err := configcatcache.CacheSegmentsFromBytes(data)
	if err != nil {
		n.log.Errorf("failed to recognise the cache format: %s", err)
		n.statusReporter.ReportError(n.sdkId, "failed to recognise the cache format")
		return false
	}
	if n.LoadEntry().ETag == eTag {
		n.statusReporter.ReportOk(n.sdkId, "config from cache not modified")
		return false
	}
	n.log.Debugf("new JSON received from cache, reloading")

	var root configcat.ConfigJson
	if err = json.Unmarshal(configJson, &root); err != nil {
		n.log.Errorf("failed to parse JSON from cache: %s", err)
		n.statusReporter.ReportError(n.sdkId, "failed to parse JSON from cache")
		return false
	}
	n.CacheEntryStore.StoreEntry(configJson, fetchTime, eTag)
	n.statusReporter.ReportOk(n.sdkId, "reload from cache succeeded")
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
