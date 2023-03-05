package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/configcat-proxy/status"
	"time"
)

type notifyingCacheStorage struct {
	poller    *time.Ticker
	stop      chan struct{}
	stored    []byte
	log       log.Logger
	reporter  status.Reporter
	ctx       context.Context
	ctxCancel func()
	store.CacheStorage
}

func NewNotifyingCacheStorage(cache store.CacheStorage, conf config.OfflineConfig, reporter status.Reporter, log log.Logger) store.Storage {
	nrLogger := log.WithPrefix("cache-poll")
	n := &notifyingCacheStorage{
		CacheStorage: cache,
		reporter:     reporter,
		log:          nrLogger,
		stop:         make(chan struct{}),
		poller:       time.NewTicker(time.Duration(conf.CachePollInterval) * time.Second),
	}
	n.ctx, n.ctxCancel = context.WithCancel(context.Background())
	n.reload()
	n.run()
	return n
}

func (n *notifyingCacheStorage) run() {
	go func() {
		for {
			select {
			case <-n.poller.C:
				if n.reload() {
					n.Notify()
				}
			case <-n.stop:
				return
			}
		}
	}()
}

func (n *notifyingCacheStorage) reload() bool {
	data, err := n.CacheStorage.Get(n.ctx, "")
	if err != nil {
		n.log.Errorf("failed to read from redis: %s", err)
		n.reporter.ReportError(status.SDK, err)
		return false
	}
	if bytes.Equal(n.stored, data) {
		n.reporter.ReportOk(status.SDK, "config from cache not modified")
		return false
	}
	n.log.Debugf("new JSON received from redis, reloading")
	var root store.RootNode
	if err = json.Unmarshal(data, &root); err != nil {
		n.log.Errorf("failed to parse JSON from redis: %s", err)
		n.reporter.ReportError(status.SDK, err)
		return false
	}
	n.stored = data
	root.Fixup()
	ser, _ := json.Marshal(root) // Re-serialize to enforce the JSON schema
	n.StoreEntry(ser)
	n.reporter.ReportOk(status.SDK, "reload from cache succeeded")
	return true
}

func (n *notifyingCacheStorage) Get(_ context.Context, _ string) ([]byte, error) {
	return n.LoadEntry().CachedJson, nil
}

func (n *notifyingCacheStorage) Set(_ context.Context, _ string, _ []byte) error {
	return nil // do nothing
}

func (n *notifyingCacheStorage) Close() {
	close(n.stop)
	n.ctxCancel()
	n.poller.Stop()
	n.CacheStorage.Close()
	n.log.Reportf("shutdown complete")
}
