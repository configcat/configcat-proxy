package file

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
	configcat "github.com/configcat/go-sdk/v9"
	"os"
	"time"
)

type watcher interface {
	Modified() <-chan struct{}
	Close()
}

type nullWatcher struct {
	modified chan struct{}
}

type fileStore struct {
	store.EntryStore
	store.Notifier

	watcher  watcher
	log      log.Logger
	conf     *config.LocalConfig
	stored   []byte
	done     chan struct{}
	reporter status.Reporter
	sdkId    string
}

var _ store.NotifyingStore = &fileStore{}

func NewFileStore(sdkId string, conf *config.LocalConfig, reporter status.Reporter, log log.Logger) configcat.ConfigCache {
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
	f := &fileStore{
		EntryStore: store.NewEntryStore(),
		Notifier:   store.NewNotifier(),
		watcher:    watch,
		log:        fileLogger,
		conf:       conf,
		reporter:   reporter,
		sdkId:      sdkId,
		done:       make(chan struct{}),
	}
	f.reload()
	go f.run()
	return f
}

func (f *fileStore) run() {
	for {
		select {
		case <-f.watcher.Modified():
			if f.reload() {
				f.Notify()
			}
		case <-f.Closed():
			return
		}
	}
}

func (f *fileStore) reload() bool {
	data, err := os.ReadFile(f.conf.FilePath)
	if err != nil {
		f.log.Errorf("failed to read file %s: %s", f.conf.FilePath, err)
		f.reporter.ReportError(f.sdkId, "failed to read file")
		return false
	}
	if bytes.Equal(f.stored, data) {
		f.reporter.ReportOk(f.sdkId, "config from file not modified")
		return false
	}
	f.log.Debugf("local JSON (%s) modified, reloading", f.conf.FilePath)
	var root configcat.ConfigJson
	if err = json.Unmarshal(data, &root); err != nil {
		f.log.Errorf("failed to parse JSON from file %s: %s", f.conf.FilePath, err)
		f.reporter.ReportError(f.sdkId, "failed to parse JSON from file")
		return false
	}
	f.stored = data
	ser, _ := json.Marshal(root) // Re-serialize to enforce the JSON schema
	f.StoreEntry(ser, time.Now().UTC(), utils.GenerateEtag(ser))
	f.reporter.ReportOk(f.sdkId, "file source reloaded")
	return true
}

func (f *fileStore) Get(_ context.Context, _ string) ([]byte, error) {
	return f.ComposeBytes(), nil
}

func (f *fileStore) Set(_ context.Context, _ string, _ []byte) error {
	return nil // do nothing
}

func (f *fileStore) Close() {
	f.Notifier.Close()
	f.watcher.Close()
	f.log.Reportf("shutdown complete")
}

func (f *nullWatcher) Close() {
	close(f.modified)
}

func (f *nullWatcher) Modified() <-chan struct{} {
	return f.modified
}
