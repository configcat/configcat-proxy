package grpc

import (
	"github.com/configcat/configcat-proxy/grpc/proto"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/stream"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type flagService struct {
	proto.UnimplementedFlagServiceServer
	streamServer stream.Server
	log          log.Logger
	sdkClient    sdk.Client
	closed       chan struct{}
}

func newFlagService(sdkClient sdk.Client, metrics metrics.Handler, log log.Logger) *flagService {
	return &flagService{
		streamServer: stream.NewServer(sdkClient, metrics, log, "grpc"),
		log:          log,
		sdkClient:    sdkClient,
		closed:       make(chan struct{}),
	}
}

func (s *flagService) EvalFlag(req *proto.Request, stream proto.FlagService_EvalFlagServer) error {
	if req.GetKey() == "" {
		return status.Error(codes.InvalidArgument, "key request parameter missing")
	}

	var user *sdk.UserAttrs
	if req.GetUser() != nil {
		user = &sdk.UserAttrs{Attrs: req.GetUser()}
	}

	conn := s.streamServer.CreateConnection(req.GetKey(), user)

	for {
		select {
		case msg := <-conn.Receive():
			payload := proto.Payload{VariationId: msg.VariationId}
			if boolVal, ok := msg.Value.(bool); ok {
				payload.Value = &proto.Payload_BoolValue{BoolValue: boolVal}
			} else if intVal, ok := msg.Value.(int); ok {
				payload.Value = &proto.Payload_IntValue{IntValue: int32(intVal)}
			} else if floatVal, ok := msg.Value.(float64); ok {
				payload.Value = &proto.Payload_DoubleValue{DoubleValue: floatVal}
			} else if stringVal, ok := msg.Value.(string); ok {
				payload.Value = &proto.Payload_StringValue{StringValue: stringVal}
			} else {
				s.log.Errorf("couldn't determine the type of '%s' for broadcasting", msg.Value)
			}
			if payload.Value != nil {
				err := stream.Send(&payload)
				if err != nil {
					s.log.Errorf("%s", err)
				}
			}
		case <-stream.Context().Done():
			s.streamServer.CloseConnection(conn, req.GetKey())
			return stream.Context().Err()
		case <-s.closed:
			return status.Error(codes.Aborted, "server down")
		}
	}
}

func (s *flagService) Close() {
	close(s.closed)
	s.streamServer.Close()
}
