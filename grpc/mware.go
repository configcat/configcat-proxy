package grpc

import (
	"context"
	"github.com/configcat/configcat-proxy/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

func DebugLogUnaryInterceptor(log log.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if isHealthCheck(info.FullMethod) {
			return handler(ctx, req)
		}

		peerCtx, ok := peer.FromContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "missing peer info")
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "missing metadata")
		}
		start := time.Now()

		log.Debugf("rpc starting %s %s %s", info.FullMethod, peerCtx.Addr, md["user-agent"])
		resp, err := handler(ctx, req)

		stat, ok := status.FromError(err)
		if !ok {
			stat = status.FromContextError(err)
		}
		duration := time.Since(start)
		log.Debugf("request finished %s %s %s [code: %s] [duration: %dms]",
			info.FullMethod, peerCtx.Addr, md["user-agent"], stat.Code().String(), duration.Milliseconds())
		return resp, err
	}
}

func DebugLogStreamInterceptor(log log.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if isHealthCheck(info.FullMethod) {
			return handler(srv, ss)
		}

		peerCtx, ok := peer.FromContext(ss.Context())
		if !ok {
			return status.Errorf(codes.InvalidArgument, "missing peer info")
		}
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Errorf(codes.InvalidArgument, "missing metadata")
		}
		start := time.Now()

		log.Debugf("rpc starting %s %s %s", info.FullMethod, peerCtx.Addr, md["user-agent"])
		err := handler(srv, ss)

		stat, ok := status.FromError(err)
		if !ok {
			stat = status.FromContextError(err)
		}
		duration := time.Since(start)
		log.Debugf("request finished %s %s %s [code: %s] [duration: %dms]",
			info.FullMethod, peerCtx.Addr, md["user-agent"], stat.Code().String(), duration.Milliseconds())
		return err
	}
}

func isHealthCheck(method string) bool {
	if strings.Contains(method, "grpc.health") {
		return true
	}
	return false
}
