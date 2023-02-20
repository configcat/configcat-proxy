package mware

import (
	"bytes"
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
	client := http.Client{}

	t.Run("with gzip", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		req.Header.Set("Accept-Encoding", "gzip")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
		var buf bytes.Buffer
		wr := gzip.NewWriter(&buf)
		_, _ = wr.Write([]byte("test"))
		_ = wr.Flush()
		assert.Equal(t, buf.Bytes(), body)
	})
	t.Run("without gzip", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, "test", string(body))
	})
}
