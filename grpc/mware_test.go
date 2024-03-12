package grpc

import (
	"bytes"
	"context"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"net"
	"testing"
)

func TestDebug_UnaryInterceptor(t *testing.T) {
	var out, errBuf bytes.Buffer
	l := log.NewLogger(&errBuf, &out, log.Debug)

	addr := net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.IPv4Mask(255, 255, 255, 255)}
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &addr})
	md := metadata.Pairs("user-agent", "test-agent")
	ctx = metadata.NewIncomingContext(ctx, md)

	handler := func(ctx context.Context, req interface{}) (i interface{}, e error) {
		return nil, nil
	}

	i := DebugLogUnaryInterceptor(l)
	_, err := i(ctx, "test-req", &grpc.UnaryServerInfo{FullMethod: "test-method"}, handler)

	assert.NoError(t, err)

	outLog := out.String()
	assert.Contains(t, outLog, "[debug] rpc starting test-method 127.0.0.1/32")
	assert.Contains(t, outLog, "[debug] request finished test-method 127.0.0.1/32 [test-agent] [code: OK] [duration: 0ms]")
}

func TestDebug_StreamInterceptor(t *testing.T) {
	var out, errBuf bytes.Buffer
	l := log.NewLogger(&errBuf, &out, log.Debug)

	addr := net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.IPv4Mask(255, 255, 255, 255)}
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &addr})
	md := metadata.Pairs("user-agent", "test-agent")
	ctx = metadata.NewIncomingContext(ctx, md)

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	i := DebugLogStreamInterceptor(l)
	err := i(ctx, MockStreamServer{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: "test-method"}, handler)

	assert.NoError(t, err)

	outLog := out.String()
	assert.Contains(t, outLog, "[debug] rpc starting test-method 127.0.0.1/32")
	assert.Contains(t, outLog, "[debug] request finished test-method 127.0.0.1/32 [test-agent] [code: OK] [duration: 0ms]")
}

func TestIsHealthCheck(t *testing.T) {
	assert.False(t, isHealthCheck("/configcat.FlagService/EvalFlag"))
	assert.True(t, isHealthCheck("/grpc.health.v1.Health/Check"))
}

type MockStreamServer struct {
	grpc.ServerStream

	ctx context.Context
}

func (s MockStreamServer) Context() context.Context {
	return s.ctx
}