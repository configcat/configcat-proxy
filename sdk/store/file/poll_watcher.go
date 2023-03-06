package file

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
	"os"
	"path/filepath"
	"time"
)

type pollWatcher struct {
	store.Notifier

	log              log.Logger
	poller           *time.Ticker
	realFilePath     string
	lastModifiedDate time.Time
	lastSize         int64
}

func newPollWatcher(conf config.LocalConfig, log log.Logger) (*pollWatcher, error) {
	fsLog := log.WithPrefix("poll-watcher")
	stat, err := os.Stat(conf.FilePath)
	if err != nil {
		fsLog.Errorf("failed to start poll watch on %s: %s", conf.FilePath, err)
		return nil, err
	}
	realPath, err := filepath.EvalSymlinks(conf.FilePath)
	if err != nil {
		fsLog.Errorf("failed to eval symlink for %s: %s", realPath, err)
		return nil, err
	}
	p := &pollWatcher{
		Notifier:         store.NewNotifier(),
		poller:           time.NewTicker(time.Duration(conf.PollInterval) * time.Second),
		log:              fsLog,
		realFilePath:     realPath,
		lastModifiedDate: stat.ModTime(),
		lastSize:         stat.Size(),
	}
	fsLog.Reportf("started watching %s", p.realFilePath)
	go p.run()
	return p, nil
}

func (p *pollWatcher) run() {
	for {
		select {
		case <-p.poller.C:
			stat, err := os.Stat(p.realFilePath)
			if err != nil {
				p.log.Errorf("failed to read stat on %s: %s", p.realFilePath, err)
				continue
			}
			if stat.ModTime() != p.lastModifiedDate || stat.Size() != p.lastSize {
				p.lastModifiedDate = stat.ModTime()
				p.lastSize = stat.Size()
				p.Notify()
			}
		case <-p.Closed():
			return
		}
	}
}

func (p *pollWatcher) Close() {
	p.Notifier.Close()
	p.poller.Stop()
	p.log.Reportf("shutdown complete")
}
