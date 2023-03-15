package grpc

import (
	"context"
	"github.com/configcat/configcat-proxy/grpc/proto"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"testing"
	"time"
)

func TestGrpc(t *testing.T) {
	sdkClient, _, _ := testutils.NewTestSdkClient(t)
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
	cl, err := client.EvalFlag(context.Background(), &proto.Request{Key: "flag", EnvId: "test"})
	assert.NoError(t, err)

	var payload *proto.Payload
	utils.WithTimeout(2*time.Second, func() {
		payload, err = cl.Recv()
		assert.NoError(t, err)
	})

	assert.True(t, payload.GetBoolValue())
}
