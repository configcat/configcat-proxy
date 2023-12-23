package web

import (
	"encoding/json"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v9/configcattest"
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
	client := http.Client{}
	req, _ := http.NewRequest(http.MethodOptions, fmt.Sprintf("%s/status", srv.URL), http.NoBody)
	resp, _ := client.Do(req)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestStatus_Get_Body(t *testing.T) {
	router := newStatusRouter(t)
	srv := httptest.NewServer(router.Handler())
	client := http.Client{}
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/status", srv.URL), http.NoBody)
	resp, _ := client.Do(req)
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
	client := http.Client{}

	t.Run("put", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/status", srv.URL), http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("post", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/status", srv.URL), http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("delete", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/status", srv.URL), http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("patch", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/status", srv.URL), http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func newStatusRouter(t *testing.T) *HttpRouter {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	ctx := testutils.NewTestSdkContext(&opts, nil)
	conf := config.Config{SDKs: map[string]*config.SDKConfig{"test": &opts}}
	reporter := status.NewReporter(&conf)
	ctx.StatusReporter = reporter
	client := sdk.NewClient(ctx, log.NewNullLogger())
	utils.WithTimeout(2*time.Second, func() {
		<-client.Ready()
	})
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return NewRouter(map[string]sdk.Client{"test": client}, nil, reporter, &config.HttpConfig{}, log.NewNullLogger())
}
