package mware

import (
	"bytes"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDebugLog(t *testing.T) {
	var out, err bytes.Buffer
	l := log.NewLogger(&err, &out, log.Debug)
	handler := DebugLog(l, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("test response"))
	})
	srv := httptest.NewServer(handler)
	client := http.Client{}

	req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
	resp, _ := client.Do(req)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	o := out.String()
	assert.Contains(t, o, "[debug] request starting HTTP/1.1 GET /")
	assert.Contains(t, o, "[debug] request finished HTTP/1.1 GET /")
	assert.Contains(t, o, "[status: 400]")
	assert.Contains(t, o, "[response: 13B]")
}
