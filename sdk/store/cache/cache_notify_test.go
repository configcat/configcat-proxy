package cache

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestRedisNotify(t *testing.T) {
	sdkKey := "key"
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	s := miniredis.RunT(t)
	red, err := newRedis(&config.RedisConfig{Addresses: []string{s.Addr()}}, log.NewNullLogger())
	assert.NoError(t, err)
	r := NewCacheStore(red, status.NewEmptyReporter())
	srv := NewNotifyingCacheStore("test", cacheKey, r, &config.OfflineConfig{CachePollInterval: 1}, status.NewEmptyReporter(), log.NewNullLogger()).(*notifyingCacheStore)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"v":{"b":true}}},"p":null}`))
	err = s.Set(cacheKey, string(cacheEntry))
	assert.NoError(t, err)
	utils.WithTimeout(2*time.Second, func() {
		<-srv.Modified()
	})
	s.CheckGet(t, cacheKey, string(cacheEntry))
	err = srv.Set(context.Background(), "", []byte{}) // set does nothing
	assert.NoError(t, err)
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_Initial(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"v":{"b":true}}},"p":null}`))
	err := s.Set(cacheKey, string(cacheEntry))
	assert.NoError(t, err)
	red, err := newRedis(&config.RedisConfig{Addresses: []string{s.Addr()}}, log.NewNullLogger())
	assert.NoError(t, err)
	r := NewCacheStore(red, status.NewEmptyReporter())
	srv := NewNotifyingCacheStore("test", cacheKey, r, &config.OfflineConfig{CachePollInterval: 1}, status.NewEmptyReporter(), log.NewNullLogger())
	s.CheckGet(t, cacheKey, string(cacheEntry))
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_Notify(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"v":{"b":false}}},"p":null}`))
	err := s.Set(cacheKey, string(cacheEntry))
	assert.NoError(t, err)
	red, err := newRedis(&config.RedisConfig{Addresses: []string{s.Addr()}}, log.NewNullLogger())
	assert.NoError(t, err)
	r := NewCacheStore(red, status.NewEmptyReporter())
	srv := NewNotifyingCacheStore("test", cacheKey, r, &config.OfflineConfig{CachePollInterval: 1}, status.NewEmptyReporter(), log.NewNullLogger()).(*notifyingCacheStore)
	s.CheckGet(t, cacheKey, string(cacheEntry))
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(r.LoadEntry().ConfigJson))

	cacheEntry = configcatcache.CacheSegmentsToBytes(time.Now(), "etag2", []byte(`{"f":{"flag":{"v":{"b":true}}},"p":null}`))
	err = s.Set(cacheKey, string(cacheEntry))
	utils.WithTimeout(2*time.Second, func() {
		<-srv.Modified()
	})
	res, err = srv.Get(context.Background(), "")
	_, _, j, _ = configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_BadJson(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	err := s.Set(cacheKey, `{"f":{"flag":{"v":{"b":false}}},"p":null}`)
	assert.NoError(t, err)
	red, err := newRedis(&config.RedisConfig{Addresses: []string{s.Addr()}}, log.NewNullLogger())
	assert.NoError(t, err)
	r := NewCacheStore(red, status.NewEmptyReporter())
	srv := NewNotifyingCacheStore("test", cacheKey, r, &config.OfflineConfig{CachePollInterval: 1}, status.NewEmptyReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_MalformedCacheEntry(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	err := s.Set(cacheKey, `{"k":{"flag`)
	assert.NoError(t, err)
	red, err := newRedis(&config.RedisConfig{Addresses: []string{s.Addr()}}, log.NewNullLogger())
	assert.NoError(t, err)
	r := NewCacheStore(red, status.NewEmptyReporter())
	srv := NewNotifyingCacheStore("test", cacheKey, r, &config.OfflineConfig{CachePollInterval: 1}, status.NewEmptyReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_MalformedJson(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"k":{"flag`))
	err := s.Set(cacheKey, string(cacheEntry))
	assert.NoError(t, err)
	red, err := newRedis(&config.RedisConfig{Addresses: []string{s.Addr()}}, log.NewNullLogger())
	assert.NoError(t, err)
	r := NewCacheStore(red, status.NewEmptyReporter())
	srv := NewNotifyingCacheStore("test", cacheKey, r, &config.OfflineConfig{CachePollInterval: 1}, status.NewEmptyReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_Reporter(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"v":{"b":true}}},"p":null}`))
	err := s.Set(cacheKey, string(cacheEntry))
	assert.NoError(t, err)
	reporter := &testReporter{}
	red, err := newRedis(&config.RedisConfig{Addresses: []string{s.Addr()}}, log.NewNullLogger())
	assert.NoError(t, err)
	r := NewCacheStore(red, reporter)
	srv := NewNotifyingCacheStore(sdkKey, cacheKey, r, &config.OfflineConfig{CachePollInterval: 1}, reporter, log.NewNullLogger()).(*notifyingCacheStore)

	rec := reporter.Records()
	assert.Contains(t, rec[len(rec)-1], "reload from cache succeeded")

	assert.Equal(t, "etag", srv.LoadEntry().ETag)
	assert.False(t, srv.reload())
	assert.Equal(t, "etag", srv.LoadEntry().ETag)

	rec = reporter.Records()
	assert.Contains(t, rec[len(rec)-1], "config from cache not modified")

	cacheEntry = configcatcache.CacheSegmentsToBytes(time.Now(), "etag2", []byte(`{"f":{"flag":{"v":`))
	err = s.Set(cacheKey, string(cacheEntry))
	assert.NoError(t, err)
	assert.False(t, srv.reload())
	assert.Equal(t, "etag", srv.LoadEntry().ETag)

	rec = reporter.Records()
	assert.Contains(t, rec[len(rec)-1], "failed to parse JSON from cache")
}

func TestRedisNotify_Unavailable(t *testing.T) {
	red, err := newRedis(&config.RedisConfig{Addresses: []string{"nonexisting"}}, log.NewNullLogger())
	assert.NoError(t, err)
	r := NewCacheStore(red, status.NewEmptyReporter())
	srv := NewNotifyingCacheStore("test", "", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewEmptyReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_Close(t *testing.T) {
	s := miniredis.RunT(t)
	red, err := newRedis(&config.RedisConfig{Addresses: []string{s.Addr()}}, log.NewNullLogger())
	assert.NoError(t, err)
	r := NewCacheStore(red, status.NewEmptyReporter())
	srv := NewNotifyingCacheStore("test", "", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewEmptyReporter(), log.NewNullLogger()).(*notifyingCacheStore)
	go func() {
		srv.Close()
	}()
	utils.WithTimeout(2*time.Second, func() {
		select {
		case <-srv.Closed():
		case <-srv.Modified():
		}
	})
}

type testReporter struct {
	records []string

	mu sync.RWMutex
}

func (r *testReporter) RegisterSdk(_ string, _ *config.SDKConfig) {
	// do nothing
}

func (r *testReporter) RemoveSdk(_ string) {
	// do nothing
}

func (r *testReporter) ReportOk(component string, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.records = append(r.records, component+"[ok] "+message)
}

func (r *testReporter) ReportError(component string, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.records = append(r.records, component+"[error] "+message)
}

func (r *testReporter) Records() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.records
}

func (r *testReporter) HttpHandler() http.HandlerFunc {
	return nil
}

func (r *testReporter) GetStatus() status.Status {
	return status.Status{}
}
