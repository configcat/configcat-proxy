package grpc

import (
	"context"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/grpc/proto"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
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

	reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: sdkSrv.URL, Key: key, PollInterval: 1}, nil)
	defer reg.Close()
	flagSrv := newFlagService(reg, nil, log.NewNullLogger())

	lis := bufconn.Listen(1024 * 1024)

	srv := grpc.NewServer()
	defer srv.GracefulStop()

	proto.RegisterFlagServiceServer(srv, flagSrv)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}))

	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	cl, err := client.EvalFlagStream(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	var payload *proto.EvalResponse
	testutils.WithTimeout(2*time.Second, func() {
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

	testutils.WithTimeout(2*time.Second, func() {
		payload, err = cl.Recv()
		assert.NoError(t, err)
	})
	assert.Equal(t, "test2", payload.GetStringValue())
}

func TestGrpc_EvalFlagStream_SdkRemoved(t *testing.T) {
	reg, h, _ := sdk.NewTestAutoRegistrarWithAutoConfig(t, config.ProfileConfig{PollInterval: 60}, log.NewNullLogger())
	flagSrv := newFlagService(reg, nil, log.NewNullLogger())

	lis := bufconn.Listen(1024 * 1024)

	srv := grpc.NewServer()
	defer srv.GracefulStop()

	proto.RegisterFlagServiceServer(srv, flagSrv)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}))

	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	cl, err := client.EvalFlagStream(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	var payload *proto.EvalResponse
	testutils.WithTimeout(2*time.Second, func() {
		payload, err = cl.Recv()
		assert.NoError(t, err)
	})
	assert.True(t, payload.GetBoolValue())

	h.RemoveSdk("test")
	reg.Refresh()
	testutils.WithTimeout(10*time.Second, func() {
		_, err = cl.Recv()
		assert.Error(t, err, "rpc error: code = Aborted desc = connection aborted")
	})
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

	reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: sdkSrv.URL, Key: key, PollInterval: 1}, nil)
	defer reg.Close()
	flagSrv := newFlagService(reg, nil, log.NewNullLogger())

	lis := bufconn.Listen(1024 * 1024)

	srv := grpc.NewServer()
	defer srv.GracefulStop()

	proto.RegisterFlagServiceServer(srv, flagSrv)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}))

	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	cl, err := client.EvalAllFlagsStream(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	var payload *proto.EvalAllResponse
	testutils.WithTimeout(2*time.Second, func() {
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

	testutils.WithTimeout(2*time.Second, func() {
		payload, err = cl.Recv()
		assert.NoError(t, err)
	})
	assert.Equal(t, 3, len(payload.GetValues()))
	assert.Equal(t, "test12", payload.GetValues()["flag1"].GetStringValue())
	assert.Equal(t, "test2", payload.GetValues()["flag2"].GetStringValue())
	assert.Equal(t, "test3", payload.GetValues()["flag3"].GetStringValue())
}

func TestGrpc_EvalAllFlagsStream_SdkRemoved(t *testing.T) {
	reg, h, _ := sdk.NewTestAutoRegistrarWithAutoConfig(t, config.ProfileConfig{PollInterval: 60}, log.NewNullLogger())
	flagSrv := newFlagService(reg, nil, log.NewNullLogger())

	lis := bufconn.Listen(1024 * 1024)

	srv := grpc.NewServer()
	defer srv.GracefulStop()

	proto.RegisterFlagServiceServer(srv, flagSrv)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}))

	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	cl, err := client.EvalAllFlagsStream(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	var payload *proto.EvalAllResponse
	testutils.WithTimeout(2*time.Second, func() {
		payload, err = cl.Recv()
		assert.NoError(t, err)
	})
	assert.Equal(t, 1, len(payload.GetValues()))
	assert.True(t, payload.GetValues()["flag"].GetBoolValue())

	h.RemoveSdk("test")
	reg.Refresh()
	testutils.WithTimeout(10*time.Second, func() {
		_, err = cl.Recv()
		assert.Error(t, err, "rpc error: code = Aborted desc = connection aborted")
	})
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

	reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: sdkSrv.URL, Key: key, PollInterval: 1}, nil)
	defer reg.Close()
	flagSrv := newFlagService(reg, nil, log.NewNullLogger())

	lis := bufconn.Listen(1024 * 1024)

	srv := grpc.NewServer()
	defer srv.GracefulStop()

	proto.RegisterFlagServiceServer(srv, flagSrv)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}))

	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	resp, err := client.EvalFlag(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	assert.Equal(t, "test1", resp.GetStringValue())

	_, err = client.EvalFlag(context.Background(), &proto.EvalRequest{Key: "non-existing", SdkId: "test", User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.Error(t, err)

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: "test2",
		},
	})

	_, err = client.Refresh(context.Background(), &proto.RefreshRequest{SdkId: "test"})
	assert.NoError(t, err)

	resp, err = client.EvalFlag(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	assert.Equal(t, "test2", resp.GetStringValue())
}

func TestGrpc_SDK_InvalidState(t *testing.T) {
	reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: "http://localhost", Key: configcattest.RandomSDKKey()}, nil)
	defer reg.Close()
	flagSrv := newFlagService(reg, nil, log.NewNullLogger())

	lis := bufconn.Listen(1024 * 1024)

	srv := grpc.NewServer()
	defer srv.GracefulStop()

	proto.RegisterFlagServiceServer(srv, flagSrv)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}))

	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)

	_, err = client.EvalFlag(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test"})
	assert.ErrorContains(t, err, "sdk with identifier 'test' is in an invalid state; please check the logs for more details")

	_, err = client.EvalAllFlags(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test"})
	assert.ErrorContains(t, err, "sdk with identifier 'test' is in an invalid state; please check the logs for more details")

	cl, err := client.EvalFlagStream(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test"})
	testutils.WithTimeout(2*time.Second, func() {
		_, err = cl.Recv()
		assert.ErrorContains(t, err, "sdk with identifier 'test' is in an invalid state; please check the logs for more details")
	})

	cl1, err := client.EvalAllFlagsStream(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test"})
	testutils.WithTimeout(2*time.Second, func() {
		_, err = cl1.Recv()
		assert.ErrorContains(t, err, "sdk with identifier 'test' is in an invalid state; please check the logs for more details")
	})

	_, err = client.GetKeys(context.Background(), &proto.KeysRequest{SdkId: "test"})
	assert.ErrorContains(t, err, "sdk with identifier 'test' is in an invalid state; please check the logs for more details")
}

func TestGrpc_Invalid_SdkKey(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
	})
	sdkSrv := httptest.NewServer(&h)
	defer sdkSrv.Close()
	reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: sdkSrv.URL, Key: key}, nil)
	defer reg.Close()
	flagSrv := newFlagService(reg, nil, log.NewNullLogger())
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	defer srv.GracefulStop()
	proto.RegisterFlagServiceServer(srv, flagSrv)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}))

	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)

	_, err = client.EvalFlag(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "non-existing"})
	assert.ErrorContains(t, err, "sdk not found for identifier: 'non-existing'")

	_, err = client.EvalAllFlags(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "non-existing"})
	assert.ErrorContains(t, err, "sdk not found for identifier: 'non-existing'")

	cl, err := client.EvalFlagStream(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "non-existing"})
	testutils.WithTimeout(2*time.Second, func() {
		_, err = cl.Recv()
		assert.ErrorContains(t, err, "sdk not found for identifier: 'non-existing'")
	})

	cl1, err := client.EvalAllFlagsStream(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "non-existing"})
	testutils.WithTimeout(2*time.Second, func() {
		_, err = cl1.Recv()
		assert.ErrorContains(t, err, "sdk not found for identifier: 'non-existing'")
	})

	_, err = client.GetKeys(context.Background(), &proto.KeysRequest{SdkId: "non-existing"})
	assert.ErrorContains(t, err, "sdk not found for identifier: 'non-existing'")

	_, err = client.Refresh(context.Background(), &proto.RefreshRequest{SdkId: "non-existing"})
	assert.ErrorContains(t, err, "sdk not found for identifier: 'non-existing'")
}

func TestGrpc_Invalid_FlagKey(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
	})
	sdkSrv := httptest.NewServer(&h)
	defer sdkSrv.Close()
	reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: sdkSrv.URL, Key: key}, nil)
	defer reg.Close()
	flagSrv := newFlagService(reg, nil, log.NewNullLogger())
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	defer srv.GracefulStop()
	proto.RegisterFlagServiceServer(srv, flagSrv)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}))

	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)

	_, err = client.EvalFlag(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test"})
	assert.ErrorContains(t, err, "feature flag or setting with key 'flag' not found")

	cl, err := client.EvalFlagStream(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test"})
	testutils.WithTimeout(2*time.Second, func() {
		_, err = cl.Recv()
		assert.ErrorContains(t, err, "feature flag or setting with key 'flag' not found")
	})
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

	reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: sdkSrv.URL, Key: key, PollInterval: 1}, nil)
	defer reg.Close()
	flagSrv := newFlagService(reg, nil, log.NewNullLogger())

	lis := bufconn.Listen(1024 * 1024)

	srv := grpc.NewServer()
	defer srv.GracefulStop()

	proto.RegisterFlagServiceServer(srv, flagSrv)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}))

	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	resp, err := client.EvalAllFlags(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
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

	resp, err = client.EvalAllFlags(context.Background(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
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

	reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: sdkSrv.URL, Key: key, PollInterval: 1}, nil)
	defer reg.Close()
	flagSrv := newFlagService(reg, nil, log.NewNullLogger())

	lis := bufconn.Listen(1024 * 1024)

	srv := grpc.NewServer()
	defer srv.GracefulStop()

	proto.RegisterFlagServiceServer(srv, flagSrv)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
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
