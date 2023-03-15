// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v4.22.0
// source: flag_service.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Request struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	EnvId string            `protobuf:"bytes,1,opt,name=env_id,json=envId,proto3" json:"env_id,omitempty"`
	Key   string            `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	User  map[string]string `protobuf:"bytes,3,rep,name=user,proto3" json:"user,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *Request) Reset() {
	*x = Request{}
	if protoimpl.UnsafeEnabled {
		mi := &file_flag_service_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Request) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Request) ProtoMessage() {}

func (x *Request) ProtoReflect() protoreflect.Message {
	mi := &file_flag_service_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Request.ProtoReflect.Descriptor instead.
func (*Request) Descriptor() ([]byte, []int) {
	return file_flag_service_proto_rawDescGZIP(), []int{0}
}

func (x *Request) GetEnvId() string {
	if x != nil {
		return x.EnvId
	}
	return ""
}

func (x *Request) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *Request) GetUser() map[string]string {
	if x != nil {
		return x.User
	}
	return nil
}

type Payload struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	VariationId string `protobuf:"bytes,1,opt,name=variation_id,json=variationId,proto3" json:"variation_id,omitempty"`
	// Types that are assignable to Value:
	//
	//	*Payload_IntValue
	//	*Payload_DoubleValue
	//	*Payload_StringValue
	//	*Payload_BoolValue
	Value isPayload_Value `protobuf_oneof:"value"`
}

func (x *Payload) Reset() {
	*x = Payload{}
	if protoimpl.UnsafeEnabled {
		mi := &file_flag_service_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Payload) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Payload) ProtoMessage() {}

func (x *Payload) ProtoReflect() protoreflect.Message {
	mi := &file_flag_service_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Payload.ProtoReflect.Descriptor instead.
func (*Payload) Descriptor() ([]byte, []int) {
	return file_flag_service_proto_rawDescGZIP(), []int{1}
}

func (x *Payload) GetVariationId() string {
	if x != nil {
		return x.VariationId
	}
	return ""
}

func (m *Payload) GetValue() isPayload_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (x *Payload) GetIntValue() int32 {
	if x, ok := x.GetValue().(*Payload_IntValue); ok {
		return x.IntValue
	}
	return 0
}

func (x *Payload) GetDoubleValue() float64 {
	if x, ok := x.GetValue().(*Payload_DoubleValue); ok {
		return x.DoubleValue
	}
	return 0
}

func (x *Payload) GetStringValue() string {
	if x, ok := x.GetValue().(*Payload_StringValue); ok {
		return x.StringValue
	}
	return ""
}

func (x *Payload) GetBoolValue() bool {
	if x, ok := x.GetValue().(*Payload_BoolValue); ok {
		return x.BoolValue
	}
	return false
}

type isPayload_Value interface {
	isPayload_Value()
}

type Payload_IntValue struct {
	IntValue int32 `protobuf:"varint,2,opt,name=int_value,json=intValue,proto3,oneof"`
}

type Payload_DoubleValue struct {
	DoubleValue float64 `protobuf:"fixed64,3,opt,name=double_value,json=doubleValue,proto3,oneof"`
}

type Payload_StringValue struct {
	StringValue string `protobuf:"bytes,4,opt,name=string_value,json=stringValue,proto3,oneof"`
}

type Payload_BoolValue struct {
	BoolValue bool `protobuf:"varint,5,opt,name=bool_value,json=boolValue,proto3,oneof"`
}

func (*Payload_IntValue) isPayload_Value() {}

func (*Payload_DoubleValue) isPayload_Value() {}

func (*Payload_StringValue) isPayload_Value() {}

func (*Payload_BoolValue) isPayload_Value() {}

var File_flag_service_proto protoreflect.FileDescriptor

var file_flag_service_proto_rawDesc = []byte{
	0x0a, 0x12, 0x66, 0x6c, 0x61, 0x67, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x63, 0x61, 0x74, 0x22,
	0x9d, 0x01, 0x0a, 0x07, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x15, 0x0a, 0x06, 0x65,
	0x6e, 0x76, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x6e, 0x76,
	0x49, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x03, 0x6b, 0x65, 0x79, 0x12, 0x30, 0x0a, 0x04, 0x75, 0x73, 0x65, 0x72, 0x18, 0x03, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x63, 0x61, 0x74, 0x2e, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x52, 0x04, 0x75, 0x73, 0x65, 0x72, 0x1a, 0x37, 0x0a, 0x09, 0x55, 0x73, 0x65, 0x72, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22,
	0xbf, 0x01, 0x0a, 0x07, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x76,
	0x61, 0x72, 0x69, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0b, 0x76, 0x61, 0x72, 0x69, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x1d,
	0x0a, 0x09, 0x69, 0x6e, 0x74, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x05, 0x48, 0x00, 0x52, 0x08, 0x69, 0x6e, 0x74, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x23, 0x0a,
	0x0c, 0x64, 0x6f, 0x75, 0x62, 0x6c, 0x65, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x01, 0x48, 0x00, 0x52, 0x0b, 0x64, 0x6f, 0x75, 0x62, 0x6c, 0x65, 0x56, 0x61, 0x6c,
	0x75, 0x65, 0x12, 0x23, 0x0a, 0x0c, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x5f, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0b, 0x73, 0x74, 0x72, 0x69,
	0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x1f, 0x0a, 0x0a, 0x62, 0x6f, 0x6f, 0x6c, 0x5f,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x08, 0x48, 0x00, 0x52, 0x09, 0x62,
	0x6f, 0x6f, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x42, 0x07, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x32, 0x45, 0x0a, 0x0b, 0x46, 0x6c, 0x61, 0x67, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x12, 0x36, 0x0a, 0x08, 0x45, 0x76, 0x61, 0x6c, 0x46, 0x6c, 0x61, 0x67, 0x12, 0x12, 0x2e, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x63, 0x61, 0x74, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x12, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x63, 0x61, 0x74, 0x2e, 0x50, 0x61, 0x79,
	0x6c, 0x6f, 0x61, 0x64, 0x22, 0x00, 0x30, 0x01, 0x42, 0x31, 0x5a, 0x2f, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x63, 0x61, 0x74,
	0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x63, 0x61, 0x74, 0x2d, 0x70, 0x72, 0x6f, 0x78, 0x79,
	0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_flag_service_proto_rawDescOnce sync.Once
	file_flag_service_proto_rawDescData = file_flag_service_proto_rawDesc
)

func file_flag_service_proto_rawDescGZIP() []byte {
	file_flag_service_proto_rawDescOnce.Do(func() {
		file_flag_service_proto_rawDescData = protoimpl.X.CompressGZIP(file_flag_service_proto_rawDescData)
	})
	return file_flag_service_proto_rawDescData
}

var file_flag_service_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_flag_service_proto_goTypes = []interface{}{
	(*Request)(nil), // 0: configcat.Request
	(*Payload)(nil), // 1: configcat.Payload
	nil,             // 2: configcat.Request.UserEntry
}
var file_flag_service_proto_depIdxs = []int32{
	2, // 0: configcat.Request.user:type_name -> configcat.Request.UserEntry
	0, // 1: configcat.FlagService.EvalFlag:input_type -> configcat.Request
	1, // 2: configcat.FlagService.EvalFlag:output_type -> configcat.Payload
	2, // [2:3] is the sub-list for method output_type
	1, // [1:2] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_flag_service_proto_init() }
func file_flag_service_proto_init() {
	if File_flag_service_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_flag_service_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Request); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_flag_service_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Payload); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_flag_service_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*Payload_IntValue)(nil),
		(*Payload_DoubleValue)(nil),
		(*Payload_StringValue)(nil),
		(*Payload_BoolValue)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_flag_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_flag_service_proto_goTypes,
		DependencyIndexes: file_flag_service_proto_depIdxs,
		MessageInfos:      file_flag_service_proto_msgTypes,
	}.Build()
	File_flag_service_proto = out.File
	file_flag_service_proto_rawDesc = nil
	file_flag_service_proto_goTypes = nil
	file_flag_service_proto_depIdxs = nil
}
