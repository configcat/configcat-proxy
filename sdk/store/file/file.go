package file

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/configcat-proxy/status"
	"os"
	"sync"
)

type watcher interface {
	Modified() <-chan struct{}
	Close()
}

type nullWatcher struct {
	modified chan struct{}
}

type fileStorage struct {
	watcher    watcher
	log        log.Logger
	conf       config.LocalConfig
	stored     []byte
	closed     chan struct{}
	closedOnce sync.Once
	reporter   status.Reporter
	*store.EntryStore
}

func NewFileStorage(conf config.LocalConfig, reporter status.Reporter, log log.Logger) store.Storage {
	fileLogger := log.WithPrefix("file-store")
	var watch watcher
	var err error
	if conf.Polling {
		watch, err = newPollWatcher(conf, fileLogger)
	} else {
		watch, err = newFileWatcher(conf, fileLogger)
		if err != nil {
			watch, err = newPollWatcher(conf, fileLogger)
		}
	}
	if err != nil {
		watch = &nullWatcher{modified: make(chan struct{})}
	}
	f := &fileStorage{
		watcher:    watch,
		log:        fileLogger,
		conf:       conf,
		reporter:   reporter,
		closed:     make(chan struct{}),
		EntryStore: store.NewEntryStore(),
	}
	f.reload()
	f.run()
	return f
}

func (f *fileStorage) run() {
	go func() {
		for {
			select {
			case <-f.watcher.Modified():
				if f.reload() {
					f.Notify()
				}
			case <-f.closed:
				f.watcher.Close()
				f.log.Reportf("shutdown complete")
				return
			}
		}
	}()
}

func (f *fileStorage) reload() bool {
	data, err := os.ReadFile(f.conf.FilePath)
	if err != nil {
		f.log.Errorf("failed to read file %s: %s", f.conf.FilePath, err)
		f.reporter.ReportError(status.SDK, err)
		return false
	}
	if bytes.Equal(f.stored, data) {
		return false
	}
	f.log.Debugf("local JSON (%s) modified, reloading", f.conf.FilePath)
	var root store.RootNode
	if err = json.Unmarshal(data, &root); err != nil {
		f.log.Errorf("failed to parse JSON from file %s: %s", f.conf.FilePath, err)
		f.reporter.ReportError(status.SDK, err)
		return false
	}
	f.stored = data
	root.Fixup()
	ser, _ := json.Marshal(root) // Re-serialize to enforce the JSON schema
	f.StoreEntry(ser)
	f.reporter.ReportOk(status.SDK, "file source reloaded")
	return true
}

func (f *fileStorage) Get(_ context.Context, _ string) ([]byte, error) {
	return f.LoadEntry().CachedJson, nil
}

func (f *fileStorage) Set(_ context.Context, _ string, _ []byte) error {
	return nil // do nothing
}

func (f *fileStorage) Close() {
	f.closedOnce.Do(func() {
		close(f.closed)
	})
}

func (f *nullWatcher) Close() {
	close(f.modified)
}

func (f *nullWatcher) Modified() <-chan struct{} {
	return f.modified
}
