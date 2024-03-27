package web

import (
	"encoding/json"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatus_Options(t *testing.T) {
	router := newStatusRouter(t)
	srv := httptest.NewServer(router.Handler())
	req, _ := http.NewRequest(http.MethodOptions, fmt.Sprintf("%s/status", srv.URL), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestStatus_Get_Body(t *testing.T) {
	router := newStatusRouter(t)
	srv := httptest.NewServer(router.Handler())
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/status", srv.URL), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	var stat status.Status
	_ = json.Unmarshal(body, &stat)

	assert.Equal(t, status.Healthy, stat.Status)
	assert.Equal(t, status.Healthy, stat.SDKs["test"].Source.Status)
	assert.Equal(t, status.Online, stat.SDKs["test"].Mode)
	assert.Equal(t, 1, len(stat.SDKs["test"].Source.Records))
	assert.Contains(t, stat.SDKs["test"].Source.Records[0], "config fetched")
	assert.Equal(t, status.RemoteSrc, stat.SDKs["test"].Source.Type)
	assert.Equal(t, status.NA, stat.Cache.Status)
	assert.Equal(t, 0, len(stat.Cache.Records))
}

func TestStatus_Not_Allowed_Methods(t *testing.T) {
	router := newStatusRouter(t)
	srv := httptest.NewServer(router.Handler())

	t.Run("put", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/status", srv.URL), http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("post", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/status", srv.URL), http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("delete", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/status", srv.URL), http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("patch", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/status", srv.URL), http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func newStatusRouter(t *testing.T) *HttpRouter {
	reporter := status.NewEmptyReporter()
	reg, _, _ := testutils.NewTestRegistrarTWithStatusReporter(t, reporter)
	client := reg.GetSdkOrNil("test")
	utils.WithTimeout(2*time.Second, func() {
		<-client.Ready()
	})
	return NewRouter(reg, nil, reporter, &config.HttpConfig{Status: config.StatusConfig{Enabled: true}}, log.NewNullLogger())
}
