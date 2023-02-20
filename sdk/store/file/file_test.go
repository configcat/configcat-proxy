package file

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestFileStore_Existing(t *testing.T) {
	utils.UseTempFile("", func(path string) {
		str := NewFileStorage(config.LocalConfig{FilePath: path}, log.NewNullLogger())
		utils.WriteIntoFile(path, `{"f":{"flag":{"v":false}},"p":null}`)
		utils.WithTimeout(2*time.Second, func() {
			<-str.Modified()
		})
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(str.GetLatestJson().CachedJson))
	})
}

func TestFileStore_Existing_Initial(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"v":false}},"p":null}`, func(path string) {
		str := NewFileStorage(config.LocalConfig{FilePath: path}, log.NewNullLogger())
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(str.GetLatestJson().CachedJson))
	})
}

func TestFileStore_Existing_Initial_Gets_MalformedJson(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"v":false}},"p":null}`, func(path string) {
		str := NewFileStorage(config.LocalConfig{FilePath: path}, log.NewNullLogger())
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(str.GetLatestJson().CachedJson))
		utils.WriteIntoFile(path, `{"f":{"flag`)
		time.Sleep(1 * time.Second)
		res, err = str.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(str.GetLatestJson().CachedJson))
	})
}

func TestFileStore_Existing_Initial_Notify(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"v":false}},"p":null}`, func(path string) {
		str := NewFileStorage(config.LocalConfig{FilePath: path}, log.NewNullLogger())
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(str.GetLatestJson().CachedJson))
		utils.WriteIntoFile(path, `{"f":{"flag":{"v":true}},"p":null}`)
		utils.WithTimeout(30*time.Second, func() {
			<-str.Modified()
		})
		res, err = str.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(res))
		assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(str.GetLatestJson().CachedJson))
	})
}

func TestFileStore_Existing_Initial_Gets_BadJson(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"v":false}},"p":null}`, func(path string) {
		str := NewFileStorage(config.LocalConfig{FilePath: path}, log.NewNullLogger())
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(str.GetLatestJson().CachedJson))
		utils.WriteIntoFile(path, `{"k":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
		time.Sleep(1 * time.Second)
		res, err = str.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, `{"f":{}}`, string(res))
		assert.Equal(t, `{"f":{}}`, string(str.GetLatestJson().CachedJson))
	})
}

func TestFileStore_Existing_Initial_BadJson(t *testing.T) {
	utils.UseTempFile(`{"k":{"flag":{"v":false}},"p":null}`, func(path string) {
		str := NewFileStorage(config.LocalConfig{FilePath: path}, log.NewNullLogger())
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, `{"f":{}}`, string(res))
		assert.Equal(t, `{"f":{}}`, string(str.GetLatestJson().CachedJson))
	})
}

func TestFileStore_Existing_Initial_MalformedJson(t *testing.T) {
	utils.UseTempFile(`{"k":{"flag`, func(path string) {
		str := NewFileStorage(config.LocalConfig{FilePath: path}, log.NewNullLogger())
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, `{"f":{}}`, string(res))
		assert.Equal(t, `{"f":{}}`, string(str.GetLatestJson().CachedJson))
	})
}

func TestFileStore_Stop(t *testing.T) {
	utils.UseTempFile("", func(path string) {
		str := NewFileStorage(config.LocalConfig{FilePath: path}, log.NewNullLogger()).(*fileStorage)
		go func() {
			str.Close()
		}()
		utils.WithTimeout(2*time.Second, func() {
			select {
			case <-str.closed:
			case <-str.Modified():
			}
		})
		res, err := str.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, `{"f":{}}`, string(res))
		assert.Equal(t, `{"f":{}}`, string(str.GetLatestJson().CachedJson))
	})
}

func TestFileStore_NonExisting(t *testing.T) {
	str := NewFileStorage(config.LocalConfig{FilePath: "nonexisting"}, log.NewNullLogger()).(*fileStorage)
	defer str.Close()

	res, err := str.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{}}`, string(res))
	assert.Equal(t, `{"f":{}}`, string(str.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{}}`, string(str.LoadEntry().CachedJson))
}
