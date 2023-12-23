package grpc

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/grpc/proto"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGrpc_EvalFlagStream(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: "test1",
		},
	})
	sdkSrv := httptest.NewServer(&h)
	defer sdkSrv.Close()

	ctx := testutils.NewTestSdkContext(&config.SDKConfig{BaseUrl: sdkSrv.URL, Key: key, PollInterval: 1}, nil)
	sdkClient := sdk.NewClient(ctx, log.NewNullLogger())
	defer sdkClient.Close()
	flagSrv := newFlagService(map[string]sdk.Client{"test": sdkClient}, nil, log.NewNullLogger())

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
	cl, err := client.EvalFlagStream(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]string{"id": "u1"}})
	assert.NoError(t, err)

	var payload *proto.EvalResponse
	utils.WithTimeout(2*time.Second, func() {
		payload, err = cl.Recv()
		assert.NoError(t, err)
	})
	assert.Equal(t, "test1", payload.GetStringValue())

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: "test2",
		},
	})

	_, err = client.Refresh(context.Background(), &proto.RefreshRequest{SdkId: "test"})
	assert.NoError(t, err)

	utils.WithTimeout(2*time.Second, func() {
		payload, err = cl.Recv()
		assert.NoError(t, err)
	})
	assert.Equal(t, "test2", payload.GetStringValue())
}

func TestGrpc_EvalAllFlagsStream(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
		"flag2": {
			Default: "test2",
		},
	})
	sdkSrv := httptest.NewServer(&h)
	defer sdkSrv.Close()

	ctx := testutils.NewTestSdkContext(&config.SDKConfig{BaseUrl: sdkSrv.URL, Key: key, PollInterval: 1}, nil)
	sdkClient := sdk.NewClient(ctx, log.NewNullLogger())
	defer sdkClient.Close()
	flagSrv := newFlagService(map[string]sdk.Client{"test": sdkClient}, nil, log.NewNullLogger())

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
	cl, err := client.EvalAllFlagsStream(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]string{"id": "u1"}})
	assert.NoError(t, err)

	var payload *proto.EvalAllResponse
	utils.WithTimeout(2*time.Second, func() {
		payload, err = cl.Recv()
		assert.NoError(t, err)
	})
	assert.Equal(t, 2, len(payload.GetValues()))
	assert.Equal(t, "test1", payload.GetValues()["flag1"].GetStringValue())
	assert.Equal(t, "test2", payload.GetValues()["flag2"].GetStringValue())

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test12",
		},
		"flag2": {
			Default: "test2",
		},
		"flag3": {
			Default: "test3",
		},
	})

	_, err = client.Refresh(context.Background(), &proto.RefreshRequest{SdkId: "test"})
	assert.NoError(t, err)

	utils.WithTimeout(2*time.Second, func() {
		payload, err = cl.Recv()
		assert.NoError(t, err)
	})
	assert.Equal(t, 2, len(payload.GetValues()))
	assert.Equal(t, "test12", payload.GetValues()["flag1"].GetStringValue())
	assert.Equal(t, "test3", payload.GetValues()["flag3"].GetStringValue())
}

func TestGrpc_EvalFlag(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: "test1",
		},
	})
	sdkSrv := httptest.NewServer(&h)
	defer sdkSrv.Close()

	ctx := testutils.NewTestSdkContext(&config.SDKConfig{BaseUrl: sdkSrv.URL, Key: key, PollInterval: 1}, nil)
	sdkClient := sdk.NewClient(ctx, log.NewNullLogger())
	defer sdkClient.Close()
	flagSrv := newFlagService(map[string]sdk.Client{"test": sdkClient}, nil, log.NewNullLogger())

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
	resp, err := client.EvalFlag(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]string{"id": "u1"}})
	assert.NoError(t, err)

	assert.Equal(t, "test1", resp.GetStringValue())

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: "test2",
		},
	})

	_, err = client.Refresh(context.Background(), &proto.RefreshRequest{SdkId: "test"})
	assert.NoError(t, err)

	resp, err = client.EvalFlag(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]string{"id": "u1"}})
	assert.NoError(t, err)

	assert.Equal(t, "test2", resp.GetStringValue())
}

func TestGrpc_EvalAllFlags(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
		"flag2": {
			Default: "test2",
		},
	})
	sdkSrv := httptest.NewServer(&h)
	defer sdkSrv.Close()

	ctx := testutils.NewTestSdkContext(&config.SDKConfig{BaseUrl: sdkSrv.URL, Key: key, PollInterval: 1}, nil)
	sdkClient := sdk.NewClient(ctx, log.NewNullLogger())
	defer sdkClient.Close()
	flagSrv := newFlagService(map[string]sdk.Client{"test": sdkClient}, nil, log.NewNullLogger())

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
	resp, err := client.EvalAllFlags(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]string{"id": "u1"}})
	assert.NoError(t, err)

	assert.Equal(t, 2, len(resp.GetValues()))
	assert.Equal(t, "test1", resp.GetValues()["flag1"].GetStringValue())
	assert.Equal(t, "test2", resp.GetValues()["flag2"].GetStringValue())

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test12",
		},
		"flag2": {
			Default: "test2",
		},
		"flag3": {
			Default: "test3",
		},
	})

	_, err = client.Refresh(context.Background(), &proto.RefreshRequest{SdkId: "test"})
	assert.NoError(t, err)

	resp, err = client.EvalAllFlags(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]string{"id": "u1"}})
	assert.NoError(t, err)

	assert.Equal(t, 3, len(resp.GetValues()))
	assert.Equal(t, "test12", resp.GetValues()["flag1"].GetStringValue())
	assert.Equal(t, "test2", resp.GetValues()["flag2"].GetStringValue())
	assert.Equal(t, "test3", resp.GetValues()["flag3"].GetStringValue())
}

func TestGrpc_GetKeys(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
		"flag2": {
			Default: "test2",
		},
	})
	sdkSrv := httptest.NewServer(&h)
	defer sdkSrv.Close()

	ctx := testutils.NewTestSdkContext(&config.SDKConfig{BaseUrl: sdkSrv.URL, Key: key, PollInterval: 1}, nil)
	sdkClient := sdk.NewClient(ctx, log.NewNullLogger())
	defer sdkClient.Close()
	flagSrv := newFlagService(map[string]sdk.Client{"test": sdkClient}, nil, log.NewNullLogger())

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
	resp, err := client.GetKeys(context.Background(), &proto.KeysRequest{SdkId: "test"})
	assert.NoError(t, err)

	assert.Equal(t, 2, len(resp.GetKeys()))
	assert.Equal(t, "flag1", resp.GetKeys()[0])
	assert.Equal(t, "flag2", resp.GetKeys()[1])

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
		"flag2": {
			Default: "test2",
		},
		"flag3": {
			Default: "test3",
		},
	})

	_, err = client.Refresh(context.Background(), &proto.RefreshRequest{SdkId: "test"})
	assert.NoError(t, err)

	resp, err = client.GetKeys(context.Background(), &proto.KeysRequest{SdkId: "test"})
	assert.NoError(t, err)

	assert.Equal(t, 3, len(resp.GetKeys()))
	assert.Equal(t, "flag1", resp.GetKeys()[0])
	assert.Equal(t, "flag2", resp.GetKeys()[1])
	assert.Equal(t, "flag3", resp.GetKeys()[2])
}
