package redis

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/configcat-proxy/status"
	"sync"
	"time"
)

type notifyingRedisStorage struct {
	poller     *time.Ticker
	closed     chan struct{}
	stored     []byte
	closedOnce sync.Once
	log        log.Logger
	ctx        context.Context
	ctxCancel  func()
	redisStorage
}

func NewNotifyingRedisStorage(sdkKey string, conf config.SDKConfig, reporter status.Reporter, log log.Logger) store.Storage {
	nrLogger := log.WithPrefix("redis-poll")
	s := newRedisStorage(sdkKey, conf.Cache.Redis, reporter)
	n := &notifyingRedisStorage{
		redisStorage: s,
		log:          nrLogger,
		closed:       make(chan struct{}),
		poller:       time.NewTicker(time.Duration(conf.Offline.CachePollInterval) * time.Second),
	}
	n.ctx, n.ctxCancel = context.WithCancel(context.Background())
	n.reload()
	n.run()
	return n
}

func (n *notifyingRedisStorage) run() {
	go func() {
		for {
			select {
			case <-n.poller.C:
				if n.reload() {
					n.Notify()
				}
			case <-n.closed:
				n.ctxCancel()
				n.poller.Stop()
				n.redisStorage.Close()
				n.log.Reportf("shutdown complete")
				return
			}
		}
	}()
}

func (n *notifyingRedisStorage) reload() bool {
	data, err := n.redisStorage.Get(n.ctx, n.cacheKey)
	if err != nil {
		n.log.Errorf("failed to read from redis: %s", err)
		n.reporter.ReportError(status.SDK, err)
		return false
	}
	if bytes.Equal(n.stored, data) {
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

func (n *notifyingRedisStorage) Get(_ context.Context, _ string) ([]byte, error) {
	return n.LoadEntry().CachedJson, nil
}

func (n *notifyingRedisStorage) Set(_ context.Context, _ string, _ []byte) error {
	return nil // do nothing
}

func (n *notifyingRedisStorage) Close() {
	n.closedOnce.Do(func() {
		close(n.closed)
	})
}
