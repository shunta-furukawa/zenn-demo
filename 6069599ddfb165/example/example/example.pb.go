// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.2
// 	protoc        v5.29.3
// source: example.proto

package example

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

type CulcRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	A             int32                  `protobuf:"varint,1,opt,name=a,proto3" json:"a,omitempty"`
	B             int32                  `protobuf:"varint,2,opt,name=b,proto3" json:"b,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CulcRequest) Reset() {
	*x = CulcRequest{}
	mi := &file_example_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CulcRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CulcRequest) ProtoMessage() {}

func (x *CulcRequest) ProtoReflect() protoreflect.Message {
	mi := &file_example_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CulcRequest.ProtoReflect.Descriptor instead.
func (*CulcRequest) Descriptor() ([]byte, []int) {
	return file_example_proto_rawDescGZIP(), []int{0}
}

func (x *CulcRequest) GetA() int32 {
	if x != nil {
		return x.A
	}
	return 0
}

func (x *CulcRequest) GetB() int32 {
	if x != nil {
		return x.B
	}
	return 0
}

type CulcResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Message       string                 `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CulcResponse) Reset() {
	*x = CulcResponse{}
	mi := &file_example_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CulcResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CulcResponse) ProtoMessage() {}

func (x *CulcResponse) ProtoReflect() protoreflect.Message {
	mi := &file_example_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CulcResponse.ProtoReflect.Descriptor instead.
func (*CulcResponse) Descriptor() ([]byte, []int) {
	return file_example_proto_rawDescGZIP(), []int{1}
}

func (x *CulcResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

var File_example_proto protoreflect.FileDescriptor

var file_example_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x22, 0x29, 0x0a, 0x0b, 0x43, 0x75, 0x6c, 0x63,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0c, 0x0a, 0x01, 0x61, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x01, 0x61, 0x12, 0x0c, 0x0a, 0x01, 0x62, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x01, 0x62, 0x22, 0x28, 0x0a, 0x0c, 0x43, 0x75, 0x6c, 0x63, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x32, 0x45, 0x0a,
	0x0e, 0x45, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12,
	0x33, 0x0a, 0x04, 0x43, 0x75, 0x6c, 0x63, 0x12, 0x14, 0x2e, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x2e, 0x43, 0x75, 0x6c, 0x63, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x15, 0x2e,
	0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x43, 0x75, 0x6c, 0x63, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x42, 0x3d, 0x5a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x73, 0x68, 0x75, 0x6e, 0x74, 0x61, 0x2d, 0x66, 0x75, 0x72, 0x75, 0x6b, 0x61,
	0x77, 0x61, 0x2f, 0x7a, 0x65, 0x6e, 0x6e, 0x2d, 0x64, 0x65, 0x6d, 0x6f, 0x2f, 0x36, 0x30, 0x36,
	0x39, 0x35, 0x39, 0x39, 0x64, 0x64, 0x66, 0x62, 0x31, 0x36, 0x35, 0x2f, 0x65, 0x78, 0x61, 0x6d,
	0x70, 0x6c, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_example_proto_rawDescOnce sync.Once
	file_example_proto_rawDescData = file_example_proto_rawDesc
)

func file_example_proto_rawDescGZIP() []byte {
	file_example_proto_rawDescOnce.Do(func() {
		file_example_proto_rawDescData = protoimpl.X.CompressGZIP(file_example_proto_rawDescData)
	})
	return file_example_proto_rawDescData
}

var file_example_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_example_proto_goTypes = []any{
	(*CulcRequest)(nil),  // 0: example.CulcRequest
	(*CulcResponse)(nil), // 1: example.CulcResponse
}
var file_example_proto_depIdxs = []int32{
	0, // 0: example.ExampleService.Culc:input_type -> example.CulcRequest
	1, // 1: example.ExampleService.Culc:output_type -> example.CulcResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_example_proto_init() }
func file_example_proto_init() {
	if File_example_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_example_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_example_proto_goTypes,
		DependencyIndexes: file_example_proto_depIdxs,
		MessageInfos:      file_example_proto_msgTypes,
	}.Build()
	File_example_proto = out.File
	file_example_proto_rawDesc = nil
	file_example_proto_goTypes = nil
	file_example_proto_depIdxs = nil
}
