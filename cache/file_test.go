package cache

import (
	"path"
	"testing"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
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
			fileCache, err := newFileStore(&config.FileCacheConfig{Folder: test.folder}, telemetry.NewEmptyReporter(), &testUserCacheDirProvider{t: t}, log.NewDebugLogger())
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

type testUserCacheDirProvider struct {
	t *testing.T
}

func (p *testUserCacheDirProvider) GetUserCacheDir() (string, error) {
	return p.t.TempDir(), nil
}
