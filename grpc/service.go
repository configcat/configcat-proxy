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
	sdkClients   map[string]sdk.Client
	closed       chan struct{}
}

func newFlagService(sdkClients map[string]sdk.Client, metrics metrics.Handler, log log.Logger) *flagService {
	return &flagService{
		streamServer: stream.NewServer(sdkClients, metrics, log, "grpc"),
		log:          log,
		sdkClients:   sdkClients,
		closed:       make(chan struct{}),
	}
}

func (s *flagService) EvalFlag(req *proto.Request, stream proto.FlagService_EvalFlagServer) error {
	if req.GetEnvId() == "" {
		return status.Error(codes.InvalidArgument, "environment id parameter missing")
	}
	if req.GetKey() == "" {
		return status.Error(codes.InvalidArgument, "key request parameter missing")
	}

	var user sdk.UserAttrs
	if req.GetUser() != nil {
		user = req.GetUser()
	}

	str := s.streamServer.GetStreamOrNil(req.GetEnvId())
	if str == nil {
		return status.Error(codes.InvalidArgument, "invalid environment identifier: '"+req.GetEnvId()+"'")
	}
	conn := str.CreateConnection(req.GetKey(), user)

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
			str.CloseConnection(conn, req.GetKey())
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
