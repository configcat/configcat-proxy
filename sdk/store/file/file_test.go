package file

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestFileStore_Existing(t *testing.T) {
	utils.UseTempFile("", func(path string) {
		str := NewFileStore("test", config.V6, &config.LocalConfig{FilePath: path}, status.NewNullReporter(), log.NewNullLogger()).(*fileStore)
		utils.WriteIntoFile(path, `{"f":{"flag":{"v":{"b":true}}},"p":null}`)
		utils.WithTimeout(2*time.Second, func() {
			<-str.Modified()
		})
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry(config.V6).ConfigJson))
	})
}

func TestFileStore_Existing_Initial(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"v":{"b":false}}},"p":null}`, func(path string) {
		str := NewFileStore("test", config.V6, &config.LocalConfig{FilePath: path}, status.NewNullReporter(), log.NewNullLogger()).(*fileStore)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry(config.V6).ConfigJson))
	})
}

func TestFileStore_Existing_Initial_Gets_MalformedJson(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"v":{"b":false}}},"p":null}`, func(path string) {
		str := NewFileStore("test", config.V6, &config.LocalConfig{FilePath: path}, status.NewNullReporter(), log.NewNullLogger()).(*fileStore)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry(config.V6).ConfigJson))
		utils.WriteIntoFile(path, `{"f":{"flag`)
		time.Sleep(1 * time.Second)
		res, err = str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ = configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry(config.V6).ConfigJson))
	})
}

func TestFileStore_Existing_Initial_Notify(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"v":{"b":false}}},"p":null}`, func(path string) {
		str := NewFileStore("test", config.V6, &config.LocalConfig{FilePath: path}, status.NewNullReporter(), log.NewNullLogger()).(*fileStore)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry(config.V6).ConfigJson))
		utils.WriteIntoFile(path, `{"f":{"flag":{"v":{"b":true}}},"p":null}`)
		utils.WithTimeout(30*time.Second, func() {
			<-str.Modified()
		})
		res, err = str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ = configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry(config.V6).ConfigJson))
	})
}

func TestFileStore_Existing_Initial_Gets_BadJson(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"v":{"b":false}}},"p":null}`, func(path string) {
		str := NewFileStore("test", config.V6, &config.LocalConfig{FilePath: path}, status.NewNullReporter(), log.NewNullLogger()).(*fileStore)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry(config.V6).ConfigJson))
		utils.WriteIntoFile(path, `{"k":{"flag":{"v":{"b":false}}},"p":null}`)
		time.Sleep(1 * time.Second)
		res, err = str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ = configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(str.LoadEntry(config.V6).ConfigJson))
	})
}

func TestFileStore_Existing_Initial_BadJson(t *testing.T) {
	utils.UseTempFile(`{"k":{"flag":{"v":{"b":false}}},"p":null}`, func(path string) {
		str := NewFileStore("test", config.V6, &config.LocalConfig{FilePath: path}, status.NewNullReporter(), log.NewNullLogger()).(*fileStore)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(str.LoadEntry(config.V6).ConfigJson))
	})
}

func TestFileStore_Existing_Initial_MalformedJson(t *testing.T) {
	utils.UseTempFile(`{"k":{"flag`, func(path string) {
		str := NewFileStore("test", config.V6, &config.LocalConfig{FilePath: path}, status.NewNullReporter(), log.NewNullLogger()).(*fileStore)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(str.LoadEntry(config.V6).ConfigJson))
	})
}

func TestFileStore_Stop(t *testing.T) {
	utils.UseTempFile("", func(path string) {
		str := NewFileStore("test", config.V6, &config.LocalConfig{FilePath: path}, status.NewNullReporter(), log.NewNullLogger()).(*fileStore)
		go func() {
			str.Close()
		}()
		utils.WithTimeout(2*time.Second, func() {
			select {
			case <-str.Closed():
			case <-str.Modified():
			}
		})
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(str.LoadEntry(config.V6).ConfigJson))
	})
}

func TestFileStore_NonExisting(t *testing.T) {
	str := NewFileStore("test", config.V6, &config.LocalConfig{FilePath: "nonexisting"}, status.NewNullReporter(), log.NewNullLogger()).(*fileStore)
	defer str.Close()

	res, err := str.Get(context.Background(), "")
	assert.NoError(t, err)
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(str.LoadEntry(config.V6).ConfigJson))
}
