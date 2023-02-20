package stream

import (
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v7/configcattest"
	"net/http/httptest"
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

	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, log.NewNullLogger())
	defer client.Close()

	strServer := NewServer(client, nil, log.NewNullLogger(), "test").(*server)
	defer strServer.Close()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sKey := fmt.Sprintf("flag%d", i)
		str := strServer.GetOrCreateStream(sKey)
		for j := 0; j < 100; j++ {
			user := sdk.UserAttrs{Attrs: map[string]string{"id": fmt.Sprintf("user%d", j)}}
			conn := str.CreateConnection(&user)
			<-conn.Receive()
			conn.Close()
		}
	}
}
