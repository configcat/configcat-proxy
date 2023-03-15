package stream

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v7/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

var connCount = 1000

func TestStreamServer_Load(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	flags := make(map[string]*configcattest.Flag)
	for i := 0; i < connCount; i++ {
		flags["flag"+strconv.Itoa(i)] = &configcattest.Flag{Default: false}
	}
	_ = h.SetFlags(key, flags)
	srv := httptest.NewServer(&h)
	defer srv.Close()

	ctx := testutils.NewTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	client := sdk.NewClient(ctx, log.NewNullLogger())
	defer client.Close()

	strServer := NewServer(map[string]sdk.Client{"test": client}, nil, log.NewNullLogger(), "test").(*server)
	defer strServer.Close()

	t.Run("init", func(t *testing.T) {
		for i := 0; i < connCount; i++ {
			fName := "flag" + strconv.Itoa(i)
			t.Run(fName, func(t *testing.T) {
				t.Parallel()
				runConnectionTest(t, fName, strServer.GetStreamOrNil("test"))
			})
		}
	})
	for i := 0; i < connCount; i++ {
		flags["flag"+strconv.Itoa(i)] = &configcattest.Flag{Default: true}
	}
	_ = h.SetFlags(key, flags)
	_ = client.Refresh()
	t.Run("check refresh", func(t *testing.T) {
		checkConnections(t, strServer)
	})
}

func runConnectionTest(t *testing.T, fName string, str Stream) {
	conn := str.CreateConnection(fName, nil)
	utils.WithTimeout(2*time.Second, func() {
		payload := <-conn.Receive()
		assert.False(t, payload.Value.(bool))
	})
}

func checkConnections(t *testing.T, srv Server) {
	str := srv.GetStreamOrNil("test").(*stream)
	for id, c := range str.channels {
		ch := c
		t.Run("chan-"+id, func(t *testing.T) {
			t.Parallel()
			for i, conn := range ch.connections {
				connect := conn
				cId := i
				t.Run("conn"+strconv.Itoa(cId), func(t *testing.T) {
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
