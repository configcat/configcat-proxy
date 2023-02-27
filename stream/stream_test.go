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

func TestStream_Connections(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	client := newClient(t, &h, key)

	stream := newStream("test", "test", client, nil, log.NewNullLogger())

	user1 := &sdk.UserAttrs{Attrs: map[string]string{"id": "u1"}}
	user2 := &sdk.UserAttrs{Attrs: map[string]string{"id": "u2"}}

	conn1 := stream.CreateConnection(user1)
	conn2 := stream.CreateConnection(user1)
	conn3 := stream.CreateConnection(user2)
	conn4 := stream.CreateConnection(nil)
	conn5 := stream.CreateConnection(nil)

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish adding connections

	assert.Same(t, conn1, stream.channels["idu1"].connections[0])
	assert.Same(t, conn2, stream.channels["idu1"].connections[1])
	assert.Same(t, conn3, stream.channels["idu2"].connections[0])
	assert.Same(t, conn4, stream.channels["no-user"].connections[0])
	assert.Same(t, conn5, stream.channels["no-user"].connections[1])

	assert.Equal(t, 3, len(stream.channels))
	assert.Equal(t, 2, len(stream.channels["idu1"].connections))
	assert.Equal(t, 1, len(stream.channels["idu2"].connections))
	assert.Equal(t, 2, len(stream.channels["no-user"].connections))

	assert.Same(t, user1, stream.channels["idu1"].user)
	assert.Same(t, user2, stream.channels["idu2"].user)
	assert.Nil(t, stream.channels["no-user"].user)

	conn2.Close()
	conn3.Close()
	conn4.Close()

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish removing connections

	assert.Equal(t, 2, len(stream.channels))
	assert.Equal(t, 1, len(stream.channels["idu1"].connections))
	assert.Nil(t, stream.channels["idu2"])
	assert.Equal(t, 1, len(stream.channels["no-user"].connections))

	conn1.Close()
	conn5.Close()

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish removing connections
	assert.Empty(t, stream.channels)
}

func TestStream_Close(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	client := newClient(t, &h, key)

	stream := newStream("test", "test", client, nil, log.NewNullLogger())

	user1 := &sdk.UserAttrs{Attrs: map[string]string{"id": "u1"}}
	user2 := &sdk.UserAttrs{Attrs: map[string]string{"id": "u2"}}

	_ = stream.CreateConnection(user1)
	_ = stream.CreateConnection(user2)

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish adding connections

	assert.Equal(t, 2, len(stream.channels))
	assert.Equal(t, 1, len(stream.channels["idu1"].connections))
	assert.Equal(t, 1, len(stream.channels["idu2"].connections))

	stream.close()

	time.Sleep(100 * time.Millisecond) // wait for goroutine finish removing connections
	assert.Empty(t, stream.channels)
}

func newClient(t *testing.T, h *configcattest.Handler, key string) sdk.Client {
	srv := httptest.NewServer(h)
	defer srv.Close()

	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
	defer client.Close()
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return client
}
