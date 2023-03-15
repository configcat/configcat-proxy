package redis

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/status"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRedisStorage(t *testing.T) {
	s := miniredis.RunT(t)
	srv := NewRedisStorage("key", &config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter()).(*redisStorage)

	err := srv.Set(context.Background(), "", []byte(`{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`))
	assert.NoError(t, err)
	s.CheckGet(t, srv.cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.LoadEntry().CachedJson))
}

func TestRedisStorage_Unavailable(t *testing.T) {
	srv := NewRedisStorage("key", &config.RedisConfig{Addresses: []string{"nonexisting"}}, status.NewNullReporter()).(*redisStorage)

	err := srv.Set(context.Background(), "", []byte(`{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`))
	assert.Error(t, err)
	_, err = srv.Get(context.Background(), "")
	assert.Error(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.LoadEntry().CachedJson))
}
