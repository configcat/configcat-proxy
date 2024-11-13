package file

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPollWatcher_Existing(t *testing.T) {
	testutils.UseTempFile("", func(path string) {
		watcher, _ := newPollWatcher(&config.LocalConfig{PollInterval: 1, FilePath: path}, log.NewNullLogger())
		testutils.WriteIntoFile(path, "test")
		testutils.WithTimeout(2*time.Second, func() {
			<-watcher.Modified()
		})
		assert.Equal(t, "test", testutils.ReadFile(path))
	})
}

func TestPollWatcher_Stop(t *testing.T) {
	testutils.UseTempFile("", func(path string) {
		watcher, _ := newPollWatcher(&config.LocalConfig{PollInterval: 1, FilePath: path}, log.NewNullLogger())
		go func() {
			watcher.Close()
		}()
		testutils.WithTimeout(2*time.Second, func() {
			select {
			case <-watcher.Closed():
			case <-watcher.Modified():
			}
		})
		assert.Equal(t, "", testutils.ReadFile(path))
	})
}

func TestPollWatcher_NonExisting(t *testing.T) {
	watcher, err := newPollWatcher(&config.LocalConfig{PollInterval: 1, FilePath: "test.txt"}, log.NewNullLogger())
	assert.Error(t, err)
	assert.Nil(t, watcher)
}
