package stream

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v9/configcattest"
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

	reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	defer reg.Close()

	strServer := NewServer(reg, nil, log.NewNullLogger(), "test").(*server)
	defer strServer.Close()

	t.Run("init", func(t *testing.T) {
		for i := 0; i < connCount; i++ {
			fName := "flag" + strconv.Itoa(i)
			t.Run(fName, func(t *testing.T) {
				t.Parallel()
				runSingleConnectionTest(t, fName, strServer.GetStreamOrNil("test"))
				runAllConnectionTest(t, fName, strServer.GetStreamOrNil("test"))
			})
		}
	})
	for i := 0; i < connCount; i++ {
		flags["flag"+strconv.Itoa(i)] = &configcattest.Flag{Default: true}
	}
	_ = h.SetFlags(key, flags)
	_ = reg.GetSdkOrNil("test").Refresh()
	assert.Equal(t, connCount, len(strServer.GetStreamOrNil("test").(*stream).channels[AllFlagsDiscriminator][0].(*allFlagsChannel).connections))
	t.Run("check refresh", func(t *testing.T) {
		checkConnections(t, strServer)
	})
}

func runSingleConnectionTest(t *testing.T, fName string, str Stream) {
	conn := str.CreateConnection(fName, nil)
	testutils.WithTimeout(2*time.Second, func() {
		payload := <-conn.Receive()
		assert.False(t, payload.(*model.ResponsePayload).Value.(bool))
	})
}

func runAllConnectionTest(t *testing.T, fName string, str Stream) {
	conn := str.CreateConnection(AllFlagsDiscriminator, nil)
	testutils.WithTimeout(2*time.Second, func() {
		payload := <-conn.Receive()
		assert.False(t, payload.(map[string]*model.ResponsePayload)[fName].Value.(bool))
	})
}

func checkConnections(t *testing.T, srv Server) {
	str := srv.GetStreamOrNil("test").(*stream)
	for id, b := range str.channels {
		bucket := b
		t.Run("chan-"+id, func(t *testing.T) {
			t.Parallel()
			for _, ch := range bucket {
				switch dChan := ch.(type) {
				case *singleFlagChannel:
					for i, conn := range dChan.connections {
						connect := conn
						cId := i
						t.Run("conn"+strconv.Itoa(cId)+"single", func(t *testing.T) {
							t.Parallel()
							testutils.WithTimeout(10*time.Second, func() {
								payload := <-connect.Receive()
								assert.True(t, payload.(*model.ResponsePayload).Value.(bool))
							})
						})
					}
				case *allFlagsChannel:
					for i, conn := range dChan.connections {
						connect := conn
						cId := i
						t.Run("conn"+strconv.Itoa(cId)+"all", func(t *testing.T) {
							t.Parallel()
							testutils.WithTimeout(2*time.Second, func() {
								payload := <-connect.Receive()
								for _, v := range payload.(map[string]*model.ResponsePayload) {
									assert.True(t, v.Value.(bool))
								}
							})
						})
					}
				}
			}
		})
	}
}
