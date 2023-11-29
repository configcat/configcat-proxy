package mware

import (
	"compress/gzip"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGZip(t *testing.T) {
	handler := GZip(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("test"))
	})
	srv := httptest.NewServer(handler)

	t.Run("with gzip", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		req.Header.Set("Accept-Encoding", "gzip")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
		gzipReader, err := gzip.NewReader(resp.Body)
		assert.NoError(t, err)
		body, _ := io.ReadAll(gzipReader)
		assert.Equal(t, "test", string(body))
	})
	t.Run("without gzip", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, "test", string(body))
	})
}
