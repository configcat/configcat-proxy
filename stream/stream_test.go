package stream

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
	"time"
)

func TestStream_Receive(t *testing.T) {
	clients, h, key := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test")
	defer str.Close()

	sConn := str.CreateConnection("flag", nil)
	aConn := str.CreateConnection(AllFlagsDiscriminator, nil)
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-sConn.Receive()
		assert.True(t, pyl.(*model.ResponsePayload).Value.(bool))
	})
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-aConn.Receive()
		assert.True(t, pyl.(map[string]*model.ResponsePayload)["flag"].Value.(bool))
	})
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: false,
		},
	})
	_ = clients["test"].Refresh()
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-sConn.Receive()
		assert.False(t, pyl.(*model.ResponsePayload).Value.(bool))
	})
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-aConn.Receive()
		assert.False(t, pyl.(map[string]*model.ResponsePayload)["flag"].Value.(bool))
	})
}

func TestStream_Offline_Receive(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true},"t":0}}}`, func(path string) {
		ctx := testutils.NewTestSdkContext(&config.SDKConfig{Key: "key", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path}}}, nil)
		client := sdk.NewClient(ctx, log.NewNullLogger())
		defer client.Close()

		str := NewStream("test", client, nil, log.NewNullLogger(), "test")
		defer str.Close()

		sConn := str.CreateConnection("flag", nil)
		aConn := str.CreateConnection(AllFlagsDiscriminator, nil)
		utils.WithTimeout(2*time.Second, func() {
			pyl := <-sConn.Receive()
			assert.True(t, pyl.(*model.ResponsePayload).Value.(bool))
		})
		utils.WithTimeout(2*time.Second, func() {
			pyl := <-aConn.Receive()
			assert.True(t, pyl.(map[string]*model.ResponsePayload)["flag"].Value.(bool))
		})
		utils.WriteIntoFile(path, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":false},"t":0}}}`)
		utils.WithTimeout(2*time.Second, func() {
			pyl := <-sConn.Receive()
			assert.False(t, pyl.(*model.ResponsePayload).Value.(bool))
		})
		utils.WithTimeout(2*time.Second, func() {
			pyl := <-aConn.Receive()
			assert.False(t, pyl.(map[string]*model.ResponsePayload)["flag"].Value.(bool))
		})
	})
}

func TestStream_Receive_Close(t *testing.T) {
	clients, _, _ := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test")
	sConn := str.CreateConnection("flag", nil)
	aConn := str.CreateConnection(AllFlagsDiscriminator, nil)
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-sConn.Receive()
		assert.True(t, pyl.(*model.ResponsePayload).Value.(bool))
	})
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-aConn.Receive()
		assert.True(t, pyl.(map[string]*model.ResponsePayload)["flag"].Value.(bool))
	})
	str.Close()
	_ = str.CreateConnection("flag", nil)
	_ = str.CreateConnection("flag", nil)
	_ = str.CreateConnection(AllFlagsDiscriminator, nil)
	_ = str.CreateConnection(AllFlagsDiscriminator, nil)
}

func TestStream_Goroutines(t *testing.T) {
	clients, _, _ := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test")
	defer str.Close()

	time.Sleep(100 * time.Millisecond)
	count := runtime.NumGoroutine()

	conn1 := str.CreateConnection("flag", nil)
	conn2 := str.CreateConnection("flag", model.UserAttrs{"id": "1"})
	conn3 := str.CreateConnection("flag", model.UserAttrs{"id": "1"})
	conn4 := str.CreateConnection("flag", model.UserAttrs{"id": "2"})
	conn5 := str.CreateConnection("flag", nil)
	conn6 := str.CreateConnection("flag", nil)
	conn7 := str.CreateConnection(AllFlagsDiscriminator, nil)
	conn8 := str.CreateConnection(AllFlagsDiscriminator, nil)
	conn9 := str.CreateConnection(AllFlagsDiscriminator, nil)

	defer func() {
		str.CloseConnection(conn1, "flag")
		str.CloseConnection(conn2, "flag")
		str.CloseConnection(conn3, "flag")
		str.CloseConnection(conn4, "flag")
		str.CloseConnection(conn5, "flag")
		str.CloseConnection(conn6, "flag")
		str.CloseConnection(conn7, AllFlagsDiscriminator)
		str.CloseConnection(conn8, AllFlagsDiscriminator)
		str.CloseConnection(conn9, AllFlagsDiscriminator)
	}()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, count, runtime.NumGoroutine())
}
