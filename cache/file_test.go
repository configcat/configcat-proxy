package cache

import (
	"errors"
	"os"
	"path"
	"testing"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
)

const (
	ErrCacheDir = iota + 1
	ErrMkdir
	ErrCreateTmp
	ErrFileWrite
	ErrFileSync
)

func TestFileCache(t *testing.T) {
	testData := []struct {
		folder string
		name   string
	}{
		{folder: "", name: "empty"},
		{folder: t.TempDir(), name: "existing"},
		{folder: path.Join(t.TempDir(), "test"), name: "non-existing"},
	}

	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			fileCache, err := newFileStore(&config.FileCacheConfig{Folder: test.folder}, telemetry.NewEmptyReporter(), &testFileSystem{t: t}, log.NewDebugLogger())
			if test.name == "non-existing" {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			cont, err := fileCache.Get(t.Context(), "cache-key")
			assert.Error(t, err)
			assert.Nil(t, cont)

			err = fileCache.Set(t.Context(), "cache-key", []byte("cache-value"))
			assert.NoError(t, err)

			cont, err = fileCache.Get(t.Context(), "cache-key")
			assert.NoError(t, err)
			assert.Equal(t, []byte("cache-value"), cont)
		})
	}
}

func TestFileCache_CacheDir_Errors(t *testing.T) {
	testData := []struct {
		folder      string
		expectedErr int
	}{
		{folder: "", expectedErr: ErrCacheDir},
		{folder: "", expectedErr: ErrMkdir},
	}

	for _, test := range testData {
		t.Run("cachedir err"+string(rune(test.expectedErr)), func(t *testing.T) {
			_, err := newFileStore(&config.FileCacheConfig{Folder: test.folder}, telemetry.NewEmptyReporter(), &testFileSystem{t: t, expectedErr: test.expectedErr}, log.NewDebugLogger())
			assert.Error(t, err)
		})
	}
}

func TestFileCache_File_Errors(t *testing.T) {
	testData := []struct {
		expectedErr     int
		shouldBeRenamed bool
	}{
		{shouldBeRenamed: true},
		{expectedErr: ErrCreateTmp},
		{expectedErr: ErrFileWrite},
		{expectedErr: ErrFileSync},
	}

	for _, test := range testData {
		t.Run("file err"+string(rune(test.expectedErr)), func(t *testing.T) {
			folder := t.TempDir()
			cache, err := newFileStore(&config.FileCacheConfig{Folder: folder}, telemetry.NewEmptyReporter(), &testFileSystem{
				t:           t,
				expectedErr: test.expectedErr,
			}, log.NewDebugLogger())
			assert.NoError(t, err)

			err = cache.Set(t.Context(), "cache-key", []byte("cache-value"))
			if test.shouldBeRenamed {
				assert.NoError(t, err)
				assert.True(t, PathExist(path.Join(folder, "cache-key.json")))
				assert.True(t, PathNotExist(path.Join(folder, "tmpfile")))
				return
			}
			assert.Error(t, err)
			assert.True(t, PathNotExist(path.Join(folder, "cache-key.json")))
			assert.True(t, PathNotExist(path.Join(folder, "tmpfile")))
		})
	}
}

type testFileSystem struct {
	t           *testing.T
	expectedErr int
}

type testFile struct {
	file        *os.File
	expectedErr int
}

func (p *testFileSystem) GetUserCacheDir() (string, error) {
	if p.expectedErr == ErrCacheDir {
		return "", errors.New("error getting user cache dir")
	}
	return p.t.TempDir(), nil
}

func (p *testFileSystem) MkdirAll(dir string, perm os.FileMode) error {
	if p.expectedErr == ErrMkdir {
		return errors.New("error creating directory")
	}
	return os.MkdirAll(dir, perm)
}

func (p *testFileSystem) CreateTemp(dir string, _ string) (fileOperations, error) {
	if p.expectedErr == ErrCreateTmp {
		return nil, errors.New("error creating temp file")
	}
	actual, _ := os.OpenFile(path.Join(dir, "tmpfile"), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	return &testFile{file: actual, expectedErr: p.expectedErr}, nil
}

func (t *testFile) Write(bytes []byte) (int, error) {
	if t.expectedErr == ErrFileWrite {
		return 0, errors.New("error writing to file")
	}
	return t.file.Write(bytes)
}

func (t *testFile) Sync() error {
	if t.expectedErr == ErrFileSync {
		return errors.New("error syncing file")
	}
	return t.file.Sync()
}

func (t *testFile) Close() error {
	return t.file.Close()
}

func (t *testFile) Name() string {
	return t.file.Name()
}

func PathExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func PathNotExist(path string) bool {
	_, err := os.Stat(path)
	return err != nil && os.IsNotExist(err)
}
