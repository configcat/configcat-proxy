package stream

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v7/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

func TestServer_Receive(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
	defer client.Close()

	strServer := NewServer(client, nil, log.NewNullLogger(), "test")
	defer strServer.Close()

	conn := strServer.CreateConnection("flag", nil)
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-conn.Receive()
		assert.True(t, pyl.Value.(bool))
	})
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: false,
		},
	})
	_ = client.Refresh()
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-conn.Receive()
		assert.False(t, pyl.Value.(bool))
	})
}

func TestServer_Offline_Receive(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, func(path string) {
		opts := config.SDKConfig{Key: "key", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path}}}
		client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
		defer client.Close()

		srv := NewServer(client, nil, log.NewNullLogger(), "test")
		defer srv.Close()

		conn := srv.CreateConnection("flag", nil)
		utils.WithTimeout(2*time.Second, func() {
			pyl := <-conn.Receive()
			assert.True(t, pyl.Value.(bool))
		})
		utils.WriteIntoFile(path, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
		utils.WithTimeout(2*time.Second, func() {
			pyl := <-conn.Receive()
			assert.False(t, pyl.Value.(bool))
		})
	})
}

func TestServer_Receive_Close(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
	defer client.Close()

	strServer := NewServer(client, nil, log.NewNullLogger(), "test").(*server)
	conn := strServer.CreateConnection("flag", nil)
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-conn.Receive()
		assert.True(t, pyl.Value.(bool))
	})
	strServer.Close()
	_ = strServer.CreateConnection("flag", nil)
	_ = strServer.CreateConnection("flag", nil)
}

func TestServer_Goroutines(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
	defer client.Close()

	strServer := NewServer(client, nil, log.NewNullLogger(), "test")
	defer strServer.Close()

	time.Sleep(100 * time.Millisecond)
	count := runtime.NumGoroutine()

	conn1 := strServer.CreateConnection("flag", nil)
	conn2 := strServer.CreateConnection("flag", &sdk.UserAttrs{Attrs: map[string]string{"id": "1"}})
	conn3 := strServer.CreateConnection("flag", &sdk.UserAttrs{Attrs: map[string]string{"id": "1"}})
	conn4 := strServer.CreateConnection("flag", &sdk.UserAttrs{Attrs: map[string]string{"id": "2"}})
	conn5 := strServer.CreateConnection("flag", nil)
	conn6 := strServer.CreateConnection("flag", nil)

	defer func() {
		strServer.CloseConnection(conn1, "flag")
		strServer.CloseConnection(conn2, "flag")
		strServer.CloseConnection(conn3, "flag")
		strServer.CloseConnection(conn4, "flag")
		strServer.CloseConnection(conn5, "flag")
		strServer.CloseConnection(conn6, "flag")
	}()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, count, runtime.NumGoroutine())
}
