package stream

import (
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/utils"
	"github.com/configcat/go-sdk/v7/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
	"time"
)

var streamCount = 10
var connCount = 50

func TestStreamServer_Load(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	flags := make(map[string]*configcattest.Flag)
	for i := 0; i < streamCount; i++ {
		flags[fmt.Sprintf("flag%d", i)] = &configcattest.Flag{Default: false}
	}
	_ = h.SetFlags(key, flags)
	srv := httptest.NewServer(&h)
	defer srv.Close()

	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, log.NewNullLogger())
	defer client.Close()

	strServer := NewServer(client, nil, log.NewNullLogger(), "test").(*server)
	defer strServer.Close()

	t.Run("init", func(t *testing.T) {
		for i := 0; i < streamCount; i++ {
			fName := fmt.Sprintf("flag%d", i)
			t.Run(fName, func(t *testing.T) {
				t.Parallel()
				runStreamTest(t, fName, strServer)
			})
		}
	})
	for i := 0; i < streamCount; i++ {
		flags[fmt.Sprintf("flag%d", i)] = &configcattest.Flag{Default: true}
	}
	_ = h.SetFlags(key, flags)
	_ = client.Refresh()
	t.Run("check refresh", func(t *testing.T) {
		for i := 0; i < streamCount; i++ {
			fName := fmt.Sprintf("flag%d", i)
			str := strServer.streams[fName]
			t.Run(fName, func(t *testing.T) {
				t.Parallel()
				checkStream(t, str)
			})
		}
	})
}

func runStreamTest(t *testing.T, fName string, strServer Server) {
	str := strServer.GetOrCreateStream(fName).(*stream)
	for i := 0; i < connCount; i++ {
		t.Run(fmt.Sprintf("conn%d", i), func(t *testing.T) {
			t.Parallel()
			runConnectionTest(t, str)
		})
	}
}

func runConnectionTest(t *testing.T, str *stream) {
	conn := str.CreateConnection(nil)
	utils.WithTimeout(2*time.Second, func() {
		payload := <-conn.Receive()
		assert.False(t, payload.Value.(bool))
	})
}

func checkStream(t *testing.T, str *stream) {
	for id, c := range str.channels {
		ch := c
		t.Run(fmt.Sprintf("chan-%s", id), func(t *testing.T) {
			t.Parallel()
			for i, conn := range ch.connections {
				connect := conn
				t.Run(fmt.Sprintf("conn%d", i), func(t *testing.T) {
					t.Parallel()
					utils.WithTimeout(2*time.Second, func() {
						payload := <-connect.Receive()
						assert.True(t, payload.Value.(bool))
					})
					connect.Close()
				})
			}
		})
	}
}