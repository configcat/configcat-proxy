package file

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
)

type fileWatcher struct {
	store.Notifier

	watch        *fsnotify.Watcher
	log          log.Logger
	realFilePath string
}

func newFileWatcher(conf config.LocalConfig, log log.Logger) (*fileWatcher, error) {
	fsLog := log.WithPrefix("file-watcher")
	_, err := os.Stat(conf.FilePath)
	if err != nil {
		fsLog.Errorf("failed to start poll watch on %s: %s", conf.FilePath, err)
		return nil, err
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		fsLog.Errorf("failed to create file watcher on %s: %s", conf.FilePath, err)
		return nil, err
	}
	dirPath := filepath.Dir(conf.FilePath)
	realPath, err := filepath.EvalSymlinks(dirPath)
	if err != nil {
		fsLog.Errorf("failed to eval symlink for %s: %s", dirPath, err)
		return nil, err
	}
	err = w.Add(realPath)
	if err != nil {
		fsLog.Errorf("failed to create file watcher on %s: %s", realPath, err)
		return nil, err
	}
	f := &fileWatcher{
		Notifier:     store.NewNotifier(),
		watch:        w,
		log:          fsLog,
		realFilePath: filepath.Join(realPath, filepath.Base(conf.FilePath)),
	}
	fsLog.Reportf("started watching %s", f.realFilePath)
	go f.run()
	return f, nil
}

func (f *fileWatcher) run() {
	for {
		select {
		case event := <-f.watch.Events:
			if event.Name == f.realFilePath && event.Has(fsnotify.Write) {
				f.Notify()
			}
		case err := <-f.watch.Errors:
			f.log.Errorf("%s", err)
		case <-f.Closed():
			return
		}
	}
}

func (f *fileWatcher) Close() {
	f.Notifier.Close()
	_ = f.watch.Close()
	f.log.Reportf("shutdown complete")
}
