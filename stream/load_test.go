package stream

import (
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v7/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
	"time"
)

var connCount = 1000

func TestStreamServer_Load(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	flags := make(map[string]*configcattest.Flag)
	for i := 0; i < connCount; i++ {
		flags[fmt.Sprintf("flag%d", i)] = &configcattest.Flag{Default: false}
	}
	_ = h.SetFlags(key, flags)
	srv := httptest.NewServer(&h)
	defer srv.Close()

	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
	defer client.Close()

	strServer := NewServer(client, nil, log.NewNullLogger(), "test").(*server)

	t.Run("init", func(t *testing.T) {
		for i := 0; i < connCount; i++ {
			fName := fmt.Sprintf("flag%d", i)
			t.Run(fName, func(t *testing.T) {
				t.Parallel()
				runConnectionTest(t, fName, strServer)
			})
		}
	})
	for i := 0; i < connCount; i++ {
		flags[fmt.Sprintf("flag%d", i)] = &configcattest.Flag{Default: true}
	}
	_ = h.SetFlags(key, flags)
	_ = client.Refresh()
	t.Run("check refresh", func(t *testing.T) {
		checkConnections(t, strServer)
	})
	strServer.Close()
}

func runConnectionTest(t *testing.T, fName string, str *server) {
	conn := str.CreateConnection(fName, nil)
	utils.WithTimeout(2*time.Second, func() {
		payload := <-conn.Receive()
		assert.False(t, payload.Value.(bool))
	})
}

func checkConnections(t *testing.T, str *server) {
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
				})
			}
		})
	}
}
