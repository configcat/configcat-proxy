package file

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestFileWatcher_Existing(t *testing.T) {
	utils.UseTempFile("", func(path string) {
		watcher, _ := newFileWatcher(config.LocalConfig{FilePath: path}, log.NewNullLogger())
		utils.WriteIntoFile(path, "test")
		utils.WithTimeout(2*time.Second, func() {
			<-watcher.Modified()
		})
		assert.Equal(t, "test", utils.ReadFile(path))
	})
}

func TestFileWatcher_Stop(t *testing.T) {
	utils.UseTempFile("", func(path string) {
		watcher, _ := newFileWatcher(config.LocalConfig{FilePath: path}, log.NewNullLogger())
		go func() {
			watcher.Close()
		}()
		utils.WithTimeout(2*time.Second, func() {
			select {
			case <-watcher.closed:
			case <-watcher.Modified():
			}
		})
		assert.Equal(t, "", utils.ReadFile(path))
	})
}

func TestFileWatcher_NonExisting(t *testing.T) {
	watcher, err := newFileWatcher(config.LocalConfig{PollInterval: 1, FilePath: "test.txt"}, log.NewNullLogger())
	assert.Error(t, err)
	assert.Nil(t, watcher)
}
