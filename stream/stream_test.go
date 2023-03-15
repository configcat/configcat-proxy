package stream

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v7/configcattest"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
	"time"
)

func TestStream_Receive(t *testing.T) {
	clients, h, key := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test")
	defer str.Close()

	conn := str.CreateConnection("flag", nil)
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-conn.Receive()
		assert.True(t, pyl.Value.(bool))
	})
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: false,
		},
	})
	_ = clients["test"].Refresh()
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-conn.Receive()
		assert.False(t, pyl.Value.(bool))
	})
}

func TestStream_Offline_Receive(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, func(path string) {
		ctx := testutils.NewTestSdkContext(&config.SDKConfig{Key: "key", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path}}}, nil)
		client := sdk.NewClient(ctx, log.NewNullLogger())
		defer client.Close()

		srv := NewStream("test", client, nil, log.NewNullLogger(), "test")
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

func TestStream_Receive_Close(t *testing.T) {
	clients, _, _ := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test")
	conn := str.CreateConnection("flag", nil)
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-conn.Receive()
		assert.True(t, pyl.Value.(bool))
	})
	str.Close()
	_ = str.CreateConnection("flag", nil)
	_ = str.CreateConnection("flag", nil)
}

func TestStream_Goroutines(t *testing.T) {
	clients, _, _ := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test")
	defer str.Close()

	time.Sleep(100 * time.Millisecond)
	count := runtime.NumGoroutine()

	conn1 := str.CreateConnection("flag", nil)
	conn2 := str.CreateConnection("flag", &sdk.UserAttrs{Attrs: map[string]string{"id": "1"}})
	conn3 := str.CreateConnection("flag", &sdk.UserAttrs{Attrs: map[string]string{"id": "1"}})
	conn4 := str.CreateConnection("flag", &sdk.UserAttrs{Attrs: map[string]string{"id": "2"}})
	conn5 := str.CreateConnection("flag", nil)
	conn6 := str.CreateConnection("flag", nil)

	defer func() {
		str.CloseConnection(conn1, "flag")
		str.CloseConnection(conn2, "flag")
		str.CloseConnection(conn3, "flag")
		str.CloseConnection(conn4, "flag")
		str.CloseConnection(conn5, "flag")
		str.CloseConnection(conn6, "flag")
	}()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, count, runtime.NumGoroutine())
}
