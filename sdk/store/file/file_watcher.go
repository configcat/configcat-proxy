package file

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"sync"
)

type fileWatcher struct {
	watch        *fsnotify.Watcher
	log          log.Logger
	closed       chan struct{}
	closedOnce   sync.Once
	modified     chan struct{}
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
		watch:        w,
		log:          fsLog,
		closed:       make(chan struct{}),
		modified:     make(chan struct{}),
		realFilePath: filepath.Join(realPath, filepath.Base(conf.FilePath)),
	}
	fsLog.Reportf("started watching %s", f.realFilePath)
	f.run()
	return f, nil
}

func (f *fileWatcher) run() {
	go func() {
		for {
			select {
			case event := <-f.watch.Events:
				if event.Name == f.realFilePath && event.Has(fsnotify.Write) {
					f.modified <- struct{}{}
				}
			case err := <-f.watch.Errors:
				f.log.Errorf("%s", err)
			case <-f.closed:
				_ = f.watch.Close()
				f.log.Reportf("shutdown complete")
				return
			}
		}
	}()
}

func (f *fileWatcher) Modified() <-chan struct{} {
	return f.modified
}

func (f *fileWatcher) Close() {
	f.closedOnce.Do(func() {
		close(f.closed)
	})
}
