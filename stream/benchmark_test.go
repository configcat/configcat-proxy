package stream

import (
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v8/configcattest"
	"net/http/httptest"
	"strconv"
	"testing"
)

func BenchmarkStream(b *testing.B) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	flags := make(map[string]*configcattest.Flag)
	for i := 0; i < b.N; i++ {
		flags[fmt.Sprintf("flag%d", i)] = &configcattest.Flag{Default: false}
	}
	_ = h.SetFlags(key, flags)
	srv := httptest.NewServer(&h)
	defer srv.Close()

	ctx := testutils.NewTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	client := sdk.NewClient(ctx, log.NewNullLogger())
	defer client.Close()

	strServer := NewServer(map[string]sdk.Client{"test": client}, nil, log.NewNullLogger(), "test").(*server)
	defer strServer.Close()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sKey := "flag" + strconv.Itoa(i)
		for j := 0; j < 100; j++ {
			user := sdk.UserAttrs{"id": "user" + strconv.Itoa(j)}
			str := strServer.GetStreamOrNil("test")
			conn := str.CreateConnection(sKey, user)
			<-conn.Receive()
			str.CloseConnection(conn, sKey)
		}
	}
}
