package grpc

import (
	"context"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/telemetry"
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
	h, key, url := newFlagServer(t, map[string]*configcattest.Flag{
		"flag": {
			Default: "test1",
		},
	})
	conn := createFlagServiceConnWithManualRegistrar(t, url, key)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	cl, err := client.EvalFlagStream(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})

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

	_, err = client.Refresh(t.Context(), &proto.RefreshRequest{Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
	assert.NoError(t, err)

	testutils.WithTimeout(2*time.Second, func() {
		payload, err = cl.Recv()
		assert.NoError(t, err)
	})
	assert.Equal(t, "test2", payload.GetStringValue())
}

func TestGrpc_EvalFlagStream_With_Sdk_Key(t *testing.T) {
	h, key, url := newFlagServer(t, map[string]*configcattest.Flag{
		"flag": {
			Default: "test1",
		},
	})
	conn := createFlagServiceConnWithManualRegistrar(t, url, key)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	cl, err := client.EvalFlagStream(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkKey{SdkKey: key}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})

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

	_, err = client.Refresh(t.Context(), &proto.RefreshRequest{Target: &proto.Target{Identifier: &proto.Target_SdkKey{SdkKey: key}}})
	assert.NoError(t, err)

	testutils.WithTimeout(2*time.Second, func() {
		payload, err = cl.Recv()
		assert.NoError(t, err)
	})
	assert.Equal(t, "test2", payload.GetStringValue())
}

func TestGrpc_EvalFlagStream_SdkRemoved(t *testing.T) {
	reg, conn, h := createFlagServiceConnWithAutoRegistrar(t)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	cl, err := client.EvalFlagStream(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
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
	h, key, url := newFlagServer(t, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
		"flag2": {
			Default: "test2",
		},
	})
	conn := createFlagServiceConnWithManualRegistrar(t, url, key)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	cl, err := client.EvalAllFlagsStream(t.Context(), &proto.EvalRequest{Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
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

	_, err = client.Refresh(t.Context(), &proto.RefreshRequest{Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
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
	reg, conn, h := createFlagServiceConnWithAutoRegistrar(t)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)

	cl, err := client.EvalAllFlagsStream(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
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
	h, key, url := newFlagServer(t, map[string]*configcattest.Flag{
		"flag": {
			Default: "test1",
		},
	})
	conn := createFlagServiceConnWithManualRegistrar(t, url, key)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	resp, err := client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	assert.Equal(t, "test1", resp.GetStringValue())

	_, err = client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "non-existing", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.Error(t, err)

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: "test2",
		},
	})

	_, err = client.Refresh(t.Context(), &proto.RefreshRequest{Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
	assert.NoError(t, err)

	resp, err = client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	assert.Equal(t, "test2", resp.GetStringValue())
}

func TestGrpc_EvalFlag_Old(t *testing.T) {
	h, key, url := newFlagServer(t, map[string]*configcattest.Flag{
		"flag": {
			Default: "test1",
		},
	})
	conn := createFlagServiceConnWithManualRegistrar(t, url, key)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	resp, err := client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	assert.Equal(t, "test1", resp.GetStringValue())

	_, err = client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "non-existing", SdkId: "test", User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.Error(t, err)

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: "test2",
		},
	})

	_, err = client.Refresh(t.Context(), &proto.RefreshRequest{SdkId: "test"})
	assert.NoError(t, err)

	resp, err = client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "flag", SdkId: "test", User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	assert.Equal(t, "test2", resp.GetStringValue())
}

func TestGrpc_EvalFlag_With_Sdk_Key(t *testing.T) {
	h, key, url := newFlagServer(t, map[string]*configcattest.Flag{
		"flag": {
			Default: "test1",
		},
	})
	conn := createFlagServiceConnWithManualRegistrar(t, url, key)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)

	resp, err := client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkKey{SdkKey: key}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	assert.Equal(t, "test1", resp.GetStringValue())

	_, err = client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "non-existing", Target: &proto.Target{Identifier: &proto.Target_SdkKey{SdkKey: key}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.Error(t, err)

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: "test2",
		},
	})

	_, err = client.Refresh(t.Context(), &proto.RefreshRequest{Target: &proto.Target{Identifier: &proto.Target_SdkKey{SdkKey: key}}})
	assert.NoError(t, err)

	resp, err = client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkKey{SdkKey: key}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	assert.Equal(t, "test2", resp.GetStringValue())
}

func TestGrpc_SDK_InvalidState(t *testing.T) {
	conn := createFlagServiceConnWithManualRegistrar(t, "http://localhost", configcattest.RandomSDKKey())
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)

	_, err := client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
	assert.ErrorContains(t, err, "requested SDK is in an invalid state; please check the logs for more details")

	_, err = client.EvalAllFlags(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
	assert.ErrorContains(t, err, "requested SDK is in an invalid state; please check the logs for more details")

	cl, err := client.EvalFlagStream(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
	testutils.WithTimeout(2*time.Second, func() {
		_, err = cl.Recv()
		assert.ErrorContains(t, err, "requested SDK is in an invalid state; please check the logs for more details")
	})

	cl1, err := client.EvalAllFlagsStream(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
	testutils.WithTimeout(2*time.Second, func() {
		_, err = cl1.Recv()
		assert.ErrorContains(t, err, "requested SDK is in an invalid state; please check the logs for more details")
	})

	_, err = client.GetKeys(t.Context(), &proto.KeysRequest{Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
	assert.ErrorContains(t, err, "requested SDK is in an invalid state; please check the logs for more details")
}

func TestGrpc_Invalid_SdkKey(t *testing.T) {
	_, key, url := newFlagServer(t, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
	})
	conn := createFlagServiceConnWithManualRegistrar(t, url, key)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)

	_, err := client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "non-existing"}}})
	assert.ErrorContains(t, err, "could not identify a configured SDK")

	_, err = client.EvalAllFlags(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "non-existing"}}})
	assert.ErrorContains(t, err, "could not identify a configured SDK")

	cl, err := client.EvalFlagStream(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "non-existing"}}})
	testutils.WithTimeout(2*time.Second, func() {
		_, err = cl.Recv()
		assert.ErrorContains(t, err, "could not identify a configured SDK")
	})

	cl1, err := client.EvalAllFlagsStream(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "non-existing"}}})
	testutils.WithTimeout(2*time.Second, func() {
		_, err = cl1.Recv()
		assert.ErrorContains(t, err, "could not identify a configured SDK")
	})

	_, err = client.GetKeys(t.Context(), &proto.KeysRequest{Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "non-existing"}}})
	assert.ErrorContains(t, err, "could not identify a configured SDK")

	_, err = client.Refresh(t.Context(), &proto.RefreshRequest{Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "non-existing"}}})
	assert.ErrorContains(t, err, "could not identify a configured SDK")
}

func TestGrpc_Invalid_Target(t *testing.T) {
	_, key, url := newFlagServer(t, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
	})
	conn := createFlagServiceConnWithManualRegistrar(t, url, key)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)

	_, err := client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "flag"})
	assert.ErrorContains(t, err, "either the sdk id or the sdk key parameter must be set")

	_, err = client.EvalAllFlags(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{}})
	assert.ErrorContains(t, err, "either the sdk id or the sdk key parameter must be set")

	cl, err := client.EvalFlagStream(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{}})
	testutils.WithTimeout(2*time.Second, func() {
		_, err = cl.Recv()
		assert.ErrorContains(t, err, "either the sdk id or the sdk key parameter must be set")
	})

	cl1, err := client.EvalAllFlagsStream(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{}})
	testutils.WithTimeout(2*time.Second, func() {
		_, err = cl1.Recv()
		assert.ErrorContains(t, err, "either the sdk id or the sdk key parameter must be set")
	})

	_, err = client.GetKeys(t.Context(), &proto.KeysRequest{Target: &proto.Target{}})
	assert.ErrorContains(t, err, "either the sdk id or the sdk key parameter must be set")

	_, err = client.Refresh(t.Context(), &proto.RefreshRequest{Target: &proto.Target{}})
	assert.ErrorContains(t, err, "either the sdk id or the sdk key parameter must be set")
}

func TestGrpc_Invalid_FlagKey(t *testing.T) {
	_, key, url := newFlagServer(t, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
	})
	conn := createFlagServiceConnWithManualRegistrar(t, url, key)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)

	_, err := client.EvalFlag(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
	assert.ErrorContains(t, err, "feature flag or setting with key 'flag' not found")

	cl, err := client.EvalFlagStream(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
	testutils.WithTimeout(2*time.Second, func() {
		_, err = cl.Recv()
		assert.ErrorContains(t, err, "feature flag or setting with key 'flag' not found")
	})
}

func TestGrpc_EvalAllFlags(t *testing.T) {
	h, key, url := newFlagServer(t, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
		"flag2": {
			Default: "test2",
		},
	})
	conn := createFlagServiceConnWithManualRegistrar(t, url, key)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	resp, err := client.EvalAllFlags(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
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

	_, err = client.Refresh(t.Context(), &proto.RefreshRequest{Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
	assert.NoError(t, err)

	resp, err = client.EvalAllFlags(t.Context(), &proto.EvalRequest{Key: "flag", Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}, User: map[string]*proto.UserValue{"id": {Value: &proto.UserValue_StringValue{StringValue: "u1"}}}})
	assert.NoError(t, err)

	assert.Equal(t, 3, len(resp.GetValues()))
	assert.Equal(t, "test12", resp.GetValues()["flag1"].GetStringValue())
	assert.Equal(t, "test2", resp.GetValues()["flag2"].GetStringValue())
	assert.Equal(t, "test3", resp.GetValues()["flag3"].GetStringValue())
}

func TestGrpc_GetKeys(t *testing.T) {
	h, key, url := newFlagServer(t, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
		"flag2": {
			Default: "test2",
		},
	})
	conn := createFlagServiceConnWithManualRegistrar(t, url, key)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	resp, err := client.GetKeys(t.Context(), &proto.KeysRequest{Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
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

	_, err = client.Refresh(t.Context(), &proto.RefreshRequest{Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
	assert.NoError(t, err)

	resp, err = client.GetKeys(t.Context(), &proto.KeysRequest{Target: &proto.Target{Identifier: &proto.Target_SdkId{SdkId: "test"}}})
	assert.NoError(t, err)

	assert.Equal(t, 3, len(resp.GetKeys()))
	assert.Equal(t, "flag1", resp.GetKeys()[0])
	assert.Equal(t, "flag2", resp.GetKeys()[1])
	assert.Equal(t, "flag3", resp.GetKeys()[2])
}

func TestGrpc_GetKeys_Old(t *testing.T) {
	h, key, url := newFlagServer(t, map[string]*configcattest.Flag{
		"flag1": {
			Default: "test1",
		},
		"flag2": {
			Default: "test2",
		},
	})
	conn := createFlagServiceConnWithManualRegistrar(t, url, key)
	defer func() {
		_ = conn.Close()
	}()

	client := proto.NewFlagServiceClient(conn)
	resp, err := client.GetKeys(t.Context(), &proto.KeysRequest{SdkId: "test"})
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

	_, err = client.Refresh(t.Context(), &proto.RefreshRequest{SdkId: "test"})
	assert.NoError(t, err)

	resp, err = client.GetKeys(t.Context(), &proto.KeysRequest{SdkId: "test"})
	assert.NoError(t, err)

	assert.Equal(t, 3, len(resp.GetKeys()))
	assert.Equal(t, "flag1", resp.GetKeys()[0])
	assert.Equal(t, "flag2", resp.GetKeys()[1])
	assert.Equal(t, "flag3", resp.GetKeys()[2])
}

func newFlagServer(t *testing.T, flags map[string]*configcattest.Flag) (h *configcattest.Handler, sdkKey string, srvUrl string) {
	key := configcattest.RandomSDKKey()
	var ha configcattest.Handler
	_ = ha.SetFlags(key, flags)
	sdkSrv := httptest.NewServer(&ha)
	t.Cleanup(func() {
		sdkSrv.Close()
	})
	return &ha, key, sdkSrv.URL
}

func createFlagServiceConnWithManualRegistrar(t *testing.T, url string, key string) *grpc.ClientConn {
	reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: url, Key: key, PollInterval: 1}, nil)
	t.Cleanup(func() {
		reg.Close()
	})
	return createFlagServiceConn(t, reg)
}

func createFlagServiceConnWithAutoRegistrar(t *testing.T) (sdk.AutoRegistrar, *grpc.ClientConn, *sdk.TestSdkRegistrarHandler) {
	reg, h, _ := sdk.NewTestAutoRegistrarWithAutoConfig(t, config.ProfileConfig{PollInterval: 60}, log.NewNullLogger())
	return reg, createFlagServiceConn(t, reg), h
}

func createFlagServiceConn(t *testing.T, registrar sdk.Registrar) *grpc.ClientConn {
	flagSrv := newFlagService(registrar, telemetry.NewEmptyReporter(), log.NewNullLogger())
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()

	proto.RegisterFlagServiceServer(srv, flagSrv)
	go func() {
		_ = srv.Serve(lis)
	}()

	conn, _ := grpc.NewClient("passthrough://bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}))

	t.Cleanup(func() {
		srv.GracefulStop()
	})

	return conn
}
