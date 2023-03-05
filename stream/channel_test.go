//go:build !race

package stream

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v7/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServer_Connections(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	client := newClient(t, &h, key)

	srv := NewServer(client, nil, log.NewNullLogger(), "test").(*server)

	user1 := &sdk.UserAttrs{Attrs: map[string]string{"id": "u1"}}
	user2 := &sdk.UserAttrs{Attrs: map[string]string{"id": "u2"}}

	conn1 := srv.CreateConnection("test", user1)
	conn2 := srv.CreateConnection("test", user1)
	conn3 := srv.CreateConnection("test", user2)
	conn4 := srv.CreateConnection("test", nil)
	conn5 := srv.CreateConnection("test", nil)

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish adding connections

	assert.Same(t, conn1, srv.channels["testidu1"].connections[0])
	assert.Same(t, conn2, srv.channels["testidu1"].connections[1])
	assert.Same(t, conn3, srv.channels["testidu2"].connections[0])
	assert.Same(t, conn4, srv.channels["test"].connections[0])
	assert.Same(t, conn5, srv.channels["test"].connections[1])

	assert.Equal(t, 3, len(srv.channels))
	assert.Equal(t, 2, len(srv.channels["testidu1"].connections))
	assert.Equal(t, 1, len(srv.channels["testidu2"].connections))
	assert.Equal(t, 2, len(srv.channels["test"].connections))

	assert.Same(t, user1, srv.channels["testidu1"].user)
	assert.Same(t, user2, srv.channels["testidu2"].user)
	assert.Nil(t, srv.channels["test"].user)

	srv.CloseConnection(conn2, "test")
	srv.CloseConnection(conn3, "test")
	srv.CloseConnection(conn4, "test")

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish removing connections

	assert.Equal(t, 2, len(srv.channels))
	assert.Equal(t, 1, len(srv.channels["testidu1"].connections))
	assert.Nil(t, srv.channels["testidu2"])
	assert.Equal(t, 1, len(srv.channels["test"].connections))

	srv.CloseConnection(conn1, "test")
	srv.CloseConnection(conn5, "test")

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish removing connections
	assert.Empty(t, srv.channels)
}

func TestServer_Close(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	client := newClient(t, &h, key)

	srv := NewServer(client, nil, log.NewNullLogger(), "test").(*server)

	user1 := &sdk.UserAttrs{Attrs: map[string]string{"id": "u1"}}
	user2 := &sdk.UserAttrs{Attrs: map[string]string{"id": "u2"}}

	_ = srv.CreateConnection("test", user1)
	_ = srv.CreateConnection("test", user2)

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish adding connections

	assert.Equal(t, 2, len(srv.channels))
	assert.Equal(t, 1, len(srv.channels["testidu1"].connections))
	assert.Equal(t, 1, len(srv.channels["testidu2"].connections))

	srv.Close()
	assert.Empty(t, srv.channels)
}

func newClient(t *testing.T, h *configcattest.Handler, key string) sdk.Client {
	srv := httptest.NewServer(h)
	defer srv.Close()

	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return client
}
