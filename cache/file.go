package cache

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type userCacheDirProvider interface {
	GetUserCacheDir() (string, error)
}

type osUserCacheDirProvider struct{}

func (p *osUserCacheDirProvider) GetUserCacheDir() (string, error) {
	return os.UserCacheDir()
}

type fileStore struct {
	folder            string
	telemetryReporter telemetry.Reporter
}

func newFileStore(conf *config.FileCacheConfig, telemetryReporter telemetry.Reporter, cacheDirProvider userCacheDirProvider, log log.Logger) (*fileStore, error) {
	fileLog := log.WithPrefix("file")
	folder := conf.Folder
	if conf.Folder == "" {
		fileLog.Reportf("cache folder is not set, using the default user cache folder")
		userCacheDir, err := cacheDirProvider.GetUserCacheDir()
		if err != nil {
			fileLog.Errorf("failed to get user cache dir: %s", err)
			return nil, fmt.Errorf("failed to get user cache dir: %s", err)
		}
		userCacheDir = path.Join(userCacheDir, "configcat-proxy")
		if _, err = os.Stat(userCacheDir); err != nil && os.IsNotExist(err) {
			err = os.MkdirAll(userCacheDir, 0700)
			if err != nil {
				fileLog.Errorf("failed to create cache folder: %s", err)
				return nil, fmt.Errorf("failed to create cache folder: %s", err)
			}
		}
		folder = userCacheDir
	} else if _, err := os.Stat(folder); err != nil && os.IsNotExist(err) {
		fileLog.Errorf("cache folder does not exist: %s", folder)
		return nil, fmt.Errorf("cache folder does not exist: %s", folder)
	}
	fileLog.Reportf("using file cache in folder %s", folder)
	return &fileStore{
		folder:            folder,
		telemetryReporter: telemetryReporter,
	}, nil
}

func (f *fileStore) Get(ctx context.Context, key string) ([]byte, error) {
	filePath := path.Join(f.folder, key+".json")
	ctx, span := f.telemetryReporter.StartSpan(ctx, "reading cache from file", semconv.FilePath(filePath))
	defer span.End()
	result, err := os.ReadFile(filePath)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, err
	}
	return result, nil
}

func (f *fileStore) Set(ctx context.Context, key string, value []byte) error {
	filePath := path.Join(f.folder, key+".json")
	ctx, span := f.telemetryReporter.StartSpan(ctx, "writing cache data to file", semconv.FilePath(filePath))
	defer span.End()

	err := f.writeAndReplaceFile(filePath, value)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return err
}

func (f *fileStore) Shutdown() {
}

func (f *fileStore) writeAndReplaceFile(filePath string, value []byte) error {
	tmpFile, err := os.CreateTemp(f.folder, filepath.Base(filePath)+"-*.tmp")
	if err != nil {
		return err
	}
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(tmpFile.Name())
		}
	}()
	if _, err = tmpFile.Write(value); err != nil {
		return err
	}
	if err = tmpFile.Sync(); err != nil {
		return err
	}
	if err = tmpFile.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpFile.Name(), filePath); err != nil {
		return err
	}
	renamed = true
	return nil
}
