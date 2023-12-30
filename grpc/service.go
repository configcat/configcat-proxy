package grpc

import (
	"context"
	"github.com/configcat/configcat-proxy/grpc/proto"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/stream"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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

func (s *flagService) EvalFlagStream(req *proto.EvalRequest, stream proto.FlagService_EvalFlagStreamServer) error {
	if req.GetSdkId() == "" {
		return status.Error(codes.InvalidArgument, "sdk id parameter missing")
	}
	if req.GetKey() == "" {
		return status.Error(codes.InvalidArgument, "key request parameter missing")
	}

	var user sdk.UserAttrs
	if req.GetUser() != nil {
		user = getUserAttrs(req.GetUser())
	}

	str := s.streamServer.GetStreamOrNil(req.GetSdkId())
	if str == nil {
		return status.Error(codes.InvalidArgument, "sdk not found for identifier: '"+req.GetSdkId()+"'")
	}
	conn := str.CreateConnection(req.GetKey(), user)

	for {
		select {
		case msg := <-conn.Receive():
			switch resp := msg.(type) {
			case *model.ResponsePayload:
				payload := s.toPayload(resp)
				if payload.Value != nil {
					err := stream.Send(payload)
					if err != nil {
						s.log.Errorf("%s", err)
					}
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

func (s *flagService) EvalAllFlagsStream(req *proto.EvalRequest, evalStream proto.FlagService_EvalAllFlagsStreamServer) error {
	if req.GetSdkId() == "" {
		return status.Error(codes.InvalidArgument, "sdk id parameter missing")
	}

	var user sdk.UserAttrs
	if req.GetUser() != nil {
		user = getUserAttrs(req.GetUser())
	}

	str := s.streamServer.GetStreamOrNil(req.GetSdkId())
	if str == nil {
		return status.Error(codes.InvalidArgument, "sdk not found for identifier: '"+req.GetSdkId()+"'")
	}
	conn := str.CreateConnection(stream.AllFlagsDiscriminator, user)

	for {
		select {
		case msg := <-conn.Receive():
			switch resp := msg.(type) {
			case map[string]*model.ResponsePayload:
				responses := make(map[string]*proto.EvalResponse)
				for key, val := range resp {
					responses[key] = s.toPayload(val)
				}
				err := evalStream.Send(&proto.EvalAllResponse{Values: responses})
				if err != nil {
					s.log.Errorf("%s", err)
				}
			}
		case <-evalStream.Context().Done():
			str.CloseConnection(conn, stream.AllFlagsDiscriminator)
			return evalStream.Context().Err()
		case <-s.closed:
			return status.Error(codes.Aborted, "server down")
		}
	}
}

func (s *flagService) EvalFlag(_ context.Context, req *proto.EvalRequest) (*proto.EvalResponse, error) {
	if req.GetSdkId() == "" {
		return nil, status.Error(codes.InvalidArgument, "sdk id parameter missing")
	}

	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key request parameter missing")
	}

	var user sdk.UserAttrs
	if req.GetUser() != nil {
		user = getUserAttrs(req.GetUser())
	}

	sdkClient, ok := s.sdkClients[req.GetSdkId()]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "sdk not found for identifier: '"+req.GetSdkId()+"'")
	}

	value, err := sdkClient.Eval(req.GetKey(), user)
	if err != nil {
		return nil, err
	}
	payload := model.PayloadFromEvalData(&value)
	return s.toPayload(&payload), nil
}

func (s *flagService) EvalAllFlags(_ context.Context, req *proto.EvalRequest) (*proto.EvalAllResponse, error) {
	if req.GetSdkId() == "" {
		return nil, status.Error(codes.InvalidArgument, "sdk id parameter missing")
	}

	var user sdk.UserAttrs
	if req.GetUser() != nil {
		user = getUserAttrs(req.GetUser())
	}

	sdkClient, ok := s.sdkClients[req.GetSdkId()]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "sdk not found for identifier: '"+req.GetSdkId()+"'")
	}

	values := sdkClient.EvalAll(user)
	final := make(map[string]*proto.EvalResponse)
	for key, value := range values {
		payload := model.PayloadFromEvalData(&value)
		final[key] = s.toPayload(&payload)
	}
	return &proto.EvalAllResponse{Values: final}, nil
}

func (s *flagService) GetKeys(_ context.Context, req *proto.KeysRequest) (*proto.KeysResponse, error) {
	if req.GetSdkId() == "" {
		return nil, status.Error(codes.InvalidArgument, "sdk id parameter missing")
	}

	sdkClient, ok := s.sdkClients[req.GetSdkId()]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "sdk not found for identifier: '"+req.GetSdkId()+"'")
	}

	keys := sdkClient.Keys()
	return &proto.KeysResponse{Keys: keys}, nil
}

func (s *flagService) Refresh(_ context.Context, req *proto.RefreshRequest) (*emptypb.Empty, error) {
	if req.GetSdkId() == "" {
		return nil, status.Error(codes.InvalidArgument, "sdk id parameter missing")
	}

	sdkClient, ok := s.sdkClients[req.GetSdkId()]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "sdk not found for identifier: '"+req.GetSdkId()+"'")
	}

	if err := sdkClient.Refresh(); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *flagService) toPayload(resp *model.ResponsePayload) *proto.EvalResponse {
	payload := &proto.EvalResponse{VariationId: resp.VariationId}
	if boolVal, ok := resp.Value.(bool); ok {
		payload.Value = &proto.EvalResponse_BoolValue{BoolValue: boolVal}
	} else if intVal, ok := resp.Value.(int); ok {
		payload.Value = &proto.EvalResponse_IntValue{IntValue: int32(intVal)}
	} else if floatVal, ok := resp.Value.(float64); ok {
		payload.Value = &proto.EvalResponse_DoubleValue{DoubleValue: floatVal}
	} else if stringVal, ok := resp.Value.(string); ok {
		payload.Value = &proto.EvalResponse_StringValue{StringValue: stringVal}
	} else {
		s.log.Errorf("couldn't determine the type of '%s' for broadcasting", resp.Value)
	}
	return payload
}

func (s *flagService) Close() {
	close(s.closed)
	s.streamServer.Close()
}

func getUserAttrs(attrs map[string]*proto.UserValue) sdk.UserAttrs {
	res := make(map[string]interface{}, len(attrs))
	for k, v := range attrs {
		if num, ok := v.GetValue().(*proto.UserValue_NumberValue); ok {
			res[k] = num.NumberValue
			continue
		}
		if str, ok := v.GetValue().(*proto.UserValue_StringValue); ok {
			res[k] = str.StringValue
			continue
		}
		if t, ok := v.GetValue().(*proto.UserValue_TimeValue); ok {
			res[k] = t.TimeValue.AsTime()
			continue
		}
		if arr, ok := v.GetValue().(*proto.UserValue_StringListValue); ok {
			res[k] = arr.StringListValue.GetValues()
			continue
		}
	}
	return res
}
