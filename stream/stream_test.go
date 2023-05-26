package stream

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
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

	sConn := str.CreateSingleFlagConnection("flag", nil)
	aConn := str.CreateAllFlagsConnection(nil)
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
	utils.UseTempFile(`{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, func(path string) {
		ctx := testutils.NewTestSdkContext(&config.SDKConfig{Key: "key", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path}}}, nil)
		client := sdk.NewClient(ctx, log.NewNullLogger())
		defer client.Close()

		str := NewStream("test", client, nil, log.NewNullLogger(), "test")
		defer str.Close()

		sConn := str.CreateSingleFlagConnection("flag", nil)
		aConn := str.CreateAllFlagsConnection(nil)
		utils.WithTimeout(2*time.Second, func() {
			pyl := <-sConn.Receive()
			assert.True(t, pyl.(*model.ResponsePayload).Value.(bool))
		})
		utils.WithTimeout(2*time.Second, func() {
			pyl := <-aConn.Receive()
			assert.True(t, pyl.(map[string]*model.ResponsePayload)["flag"].Value.(bool))
		})
		utils.WriteIntoFile(path, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
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
	sConn := str.CreateSingleFlagConnection("flag", nil)
	aConn := str.CreateAllFlagsConnection(nil)
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-sConn.Receive()
		assert.True(t, pyl.(*model.ResponsePayload).Value.(bool))
	})
	utils.WithTimeout(2*time.Second, func() {
		pyl := <-aConn.Receive()
		assert.True(t, pyl.(map[string]*model.ResponsePayload)["flag"].Value.(bool))
	})
	str.Close()
	_ = str.CreateSingleFlagConnection("flag", nil)
	_ = str.CreateSingleFlagConnection("flag", nil)
	_ = str.CreateAllFlagsConnection(nil)
	_ = str.CreateAllFlagsConnection(nil)
}

func TestStream_Goroutines(t *testing.T) {
	clients, _, _ := testutils.NewTestSdkClient(t)

	str := NewStream("test", clients["test"], nil, log.NewNullLogger(), "test")
	defer str.Close()

	time.Sleep(100 * time.Millisecond)
	count := runtime.NumGoroutine()

	conn1 := str.CreateSingleFlagConnection("flag", nil)
	conn2 := str.CreateSingleFlagConnection("flag", sdk.UserAttrs{"id": "1"})
	conn3 := str.CreateSingleFlagConnection("flag", sdk.UserAttrs{"id": "1"})
	conn4 := str.CreateSingleFlagConnection("flag", sdk.UserAttrs{"id": "2"})
	conn5 := str.CreateSingleFlagConnection("flag", nil)
	conn6 := str.CreateSingleFlagConnection("flag", nil)
	conn7 := str.CreateAllFlagsConnection(nil)
	conn8 := str.CreateAllFlagsConnection(nil)
	conn9 := str.CreateAllFlagsConnection(nil)

	defer func() {
		str.CloseSingleFlagConnection(conn1, "flag")
		str.CloseSingleFlagConnection(conn2, "flag")
		str.CloseSingleFlagConnection(conn3, "flag")
		str.CloseSingleFlagConnection(conn4, "flag")
		str.CloseSingleFlagConnection(conn5, "flag")
		str.CloseSingleFlagConnection(conn6, "flag")
		str.CloseAllFlagsConnection(conn7)
		str.CloseAllFlagsConnection(conn8)
		str.CloseAllFlagsConnection(conn9)
	}()

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, count, runtime.NumGoroutine())
}
