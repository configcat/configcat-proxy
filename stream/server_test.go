package stream

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/configcat-proxy/utils"
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
	stream := strServer.GetOrCreateStream("flag")

	assert.Same(t, stream, strServer.GetOrCreateStream("flag"))

	conn := stream.CreateConnection(nil)
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
		stream := srv.GetOrCreateStream("flag")

		assert.Same(t, stream, srv.GetOrCreateStream("flag"))

		conn := stream.CreateConnection(nil)
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

func TestServer_Stream_Stale_Close(t *testing.T) {
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
	defer strServer.Close()
	stream := strServer.GetOrCreateStream("flag").(*stream)

	conn := stream.CreateConnection(nil)
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-conn.Receive()
		assert.True(t, pyl.Value.(bool))
	})
	conn.Close()
	time.Sleep(100 * time.Millisecond)
	assert.True(t, stream.markedForClose.Load())
	strServer.teardownStaleStreams()

	assert.Empty(t, strServer.streams)
}

func TestServer_Stream_NotStale(t *testing.T) {
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
	defer strServer.Close()
	stream := strServer.GetOrCreateStream("flag").(*stream)

	conn := stream.CreateConnection(nil)
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-conn.Receive()
		assert.True(t, pyl.Value.(bool))
	})
	strServer.teardownStaleStreams()

	assert.NotEmpty(t, strServer.streams)
}

func TestServer_Stream_TearDown_All(t *testing.T) {
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
	defer strServer.Close()
	stream := strServer.GetOrCreateStream("flag").(*stream)

	conn := stream.CreateConnection(nil)
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-conn.Receive()
		assert.True(t, pyl.Value.(bool))
	})
	strServer.teardownAllStreams()

	assert.Empty(t, strServer.streams)
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

	stream := strServer.GetOrCreateStream("flag")
	stream = strServer.GetOrCreateStream("flag")
	stream = strServer.GetOrCreateStream("flag")
	stream = strServer.GetOrCreateStream("flag")
	stream = strServer.GetOrCreateStream("flag")
	stream = strServer.GetOrCreateStream("flag")
	conn1 := stream.CreateConnection(nil)
	conn2 := stream.CreateConnection(&sdk.UserAttrs{Attrs: map[string]string{"id": "1"}})
	conn3 := stream.CreateConnection(&sdk.UserAttrs{Attrs: map[string]string{"id": "1"}})
	conn4 := stream.CreateConnection(&sdk.UserAttrs{Attrs: map[string]string{"id": "2"}})
	conn5 := stream.CreateConnection(nil)
	conn6 := stream.CreateConnection(nil)

	defer func() {
		conn1.Close()
		conn2.Close()
		conn3.Close()
		conn4.Close()
		conn5.Close()
		conn6.Close()
	}()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, count+1, runtime.NumGoroutine())
}
