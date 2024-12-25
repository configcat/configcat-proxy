package file

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestFileStore_Existing(t *testing.T) {
	testutils.UseTempFile("", func(path string) {
		str := NewFileStore("test", &config.LocalConfig{FilePath: path}, status.NewEmptyReporter(), log.NewNullLogger()).(*fileStore)
		testutils.WriteIntoFile(path, `{"f":{"flag":{"v":{"b":true}}},"p":null}`)
		testutils.WithTimeout(2*time.Second, func() {
			<-str.Modified()
		})
		err := str.Set(context.Background(), "", []byte{}) // set does nothing
		assert.NoError(t, err)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry().ConfigJson))
	})
}

func TestFileStore_Existing_Initial(t *testing.T) {
	testutils.UseTempFile(`{"f":{"flag":{"v":{"b":false}}},"p":null}`, func(path string) {
		str := NewFileStore("test", &config.LocalConfig{FilePath: path}, status.NewEmptyReporter(), log.NewNullLogger()).(*fileStore)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry().ConfigJson))
	})
}

func TestFileStore_Existing_Initial_Gets_MalformedJson(t *testing.T) {
	testutils.UseTempFile(`{"f":{"flag":{"v":{"b":false}}},"p":null}`, func(path string) {
		str := NewFileStore("test", &config.LocalConfig{FilePath: path}, status.NewEmptyReporter(), log.NewNullLogger()).(*fileStore)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry().ConfigJson))
		testutils.WriteIntoFile(path, `{"f":{"flag`)
		time.Sleep(1 * time.Second)
		res, err = str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ = configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry().ConfigJson))
	})
}

func TestFileStore_Existing_Initial_Notify(t *testing.T) {
	testutils.UseTempFile(`{"f":{"flag":{"v":{"b":false}}},"p":null}`, func(path string) {
		str := NewFileStore("test", &config.LocalConfig{FilePath: path}, status.NewEmptyReporter(), log.NewNullLogger()).(*fileStore)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry().ConfigJson))
		testutils.WriteIntoFile(path, `{"f":{"flag":{"v":{"b":true}}},"p":null}`)
		testutils.WithTimeout(30*time.Second, func() {
			<-str.Modified()
		})
		res, err = str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ = configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry().ConfigJson))
	})
}

func TestFileStore_Existing_Initial_Gets_BadJson(t *testing.T) {
	testutils.UseTempFile(`{"f":{"flag":{"v":{"b":false}}},"p":null}`, func(path string) {
		str := NewFileStore("test", &config.LocalConfig{FilePath: path}, status.NewEmptyReporter(), log.NewNullLogger()).(*fileStore)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(str.LoadEntry().ConfigJson))
		testutils.WriteIntoFile(path, `{"k":{"flag":{"v":{"b":false}}},"p":null}`)
		time.Sleep(1 * time.Second)
		res, err = str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ = configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(str.LoadEntry().ConfigJson))
	})
}

func TestFileStore_Existing_Initial_BadJson(t *testing.T) {
	testutils.UseTempFile(`{"k":{"flag":{"v":{"b":false}}},"p":null}`, func(path string) {
		str := NewFileStore("test", &config.LocalConfig{FilePath: path}, status.NewEmptyReporter(), log.NewNullLogger()).(*fileStore)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(str.LoadEntry().ConfigJson))
	})
}

func TestFileStore_Existing_Initial_MalformedJson(t *testing.T) {
	testutils.UseTempFile(`{"k":{"flag`, func(path string) {
		str := NewFileStore("test", &config.LocalConfig{FilePath: path}, status.NewEmptyReporter(), log.NewNullLogger()).(*fileStore)
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(str.LoadEntry().ConfigJson))
	})
}

func TestFileStore_Stop(t *testing.T) {
	testutils.UseTempFile("", func(path string) {
		str := NewFileStore("test", &config.LocalConfig{FilePath: path}, status.NewEmptyReporter(), log.NewNullLogger()).(*fileStore)
		go func() {
			str.Close()
		}()
		testutils.WithTimeout(2*time.Second, func() {
			select {
			case <-str.Context().Done():
			case <-str.Modified():
			}
		})
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(str.LoadEntry().ConfigJson))
	})
}

func TestFileStore_NonExisting(t *testing.T) {
	str := NewFileStore("test", &config.LocalConfig{FilePath: "nonexisting"}, status.NewEmptyReporter(), log.NewNullLogger()).(*fileStore)
	defer str.Close()

	res, err := str.Get(context.Background(), "")
	assert.NoError(t, err)
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(str.LoadEntry().ConfigJson))
}
