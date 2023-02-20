package mware

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeaders(t *testing.T) {
	handler := ExtraHeaders(map[string]string{"h1": "v1"}, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(handler)
	client := http.Client{}
	req, _ := http.NewRequest(http.MethodOptions, srv.URL, http.NoBody)
	resp, _ := client.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "v1", resp.Header.Get("h1"))
}
