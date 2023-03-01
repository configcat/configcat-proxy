package grpc

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/grpc/proto"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v7/configcattest"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGrpc(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h = &configcattest.Handler{}
	_ = h.SetFlags(key, map[string]*configcattest.Flag{"flag": {Default: true}})
	sdkClient := newClient(t, h, key)
	flagSrv := newFlagService(sdkClient, nil, log.NewNullLogger())

	lis := bufconn.Listen(1024 * 1024)

	srv := grpc.NewServer()
	defer srv.GracefulStop()

	proto.RegisterFlagServiceServer(srv, flagSrv)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.DialContext(context.Background(),
		"bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}))

	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	cl, err := client.EvalFlag(context.Background(), &proto.Request{Key: "flag"})
	assert.NoError(t, err)

	var payload *proto.Payload
	utils.WithTimeout(2*time.Second, func() {
		payload, err = cl.Recv()
		assert.NoError(t, err)
	})

	assert.True(t, payload.GetBoolValue())
}

func newClient(t *testing.T, h *configcattest.Handler, key string) sdk.Client {
	srv := httptest.NewServer(h)
	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return client
}
