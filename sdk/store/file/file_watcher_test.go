package file

import (
	"testing"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
)

func TestFileWatcher_Existing(t *testing.T) {
	testutils.UseTempFile("", func(path string) {
		watcher, _ := newFileWatcher(&config.LocalConfig{FilePath: path}, log.NewNullLogger())
		testutils.WriteIntoFile(path, "test")
		testutils.WithTimeout(2*time.Second, func() {
			<-watcher.Modified()
		})
		assert.Equal(t, "test", testutils.ReadFile(path))
	})
}

func TestFileWatcher_Stop(t *testing.T) {
	testutils.UseTempFile("", func(path string) {
		watcher, _ := newFileWatcher(&config.LocalConfig{FilePath: path}, log.NewNullLogger())
		go func() {
			watcher.Close()
		}()
		testutils.WithTimeout(2*time.Second, func() {
			select {
			case <-watcher.Context().Done():
			case <-watcher.Modified():
			}
		})
		assert.Equal(t, "", testutils.ReadFile(path))
	})
}

func TestFileWatcher_NonExisting(t *testing.T) {
	watcher, err := newFileWatcher(&config.LocalConfig{PollInterval: 1, FilePath: "test.txt"}, log.NewNullLogger())
	assert.Error(t, err)
	assert.Nil(t, watcher)
}
