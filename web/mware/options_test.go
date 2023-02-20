package mware

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOptions(t *testing.T) {
	handler := AutoOptions(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
	})
	srv := httptest.NewServer(handler)
	client := http.Client{}

	t.Run("options", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, srv.URL, http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
	t.Run("get", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
