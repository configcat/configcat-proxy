package file

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPollWatcher_Existing(t *testing.T) {
	utils.UseTempFile("", func(path string) {
		watcher, _ := newPollWatcher(config.LocalConfig{PollInterval: 1, FilePath: path}, log.NewNullLogger())
		utils.WriteIntoFile(path, "test")
		utils.WithTimeout(2*time.Second, func() {
			<-watcher.Modified()
		})
		assert.Equal(t, "test", utils.ReadFile(path))
	})
}

func TestPollWatcher_Stop(t *testing.T) {
	utils.UseTempFile("", func(path string) {
		watcher, _ := newPollWatcher(config.LocalConfig{PollInterval: 1, FilePath: path}, log.NewNullLogger())
		go func() {
			watcher.Close()
		}()
		utils.WithTimeout(2*time.Second, func() {
			select {
			case <-watcher.stop:
			case <-watcher.Modified():
			}
		})
		assert.Equal(t, "", utils.ReadFile(path))
	})
}

func TestPollWatcher_NonExisting(t *testing.T) {
	watcher, err := newPollWatcher(config.LocalConfig{PollInterval: 1, FilePath: "test.txt"}, log.NewNullLogger())
	assert.Error(t, err)
	assert.Nil(t, watcher)
}
