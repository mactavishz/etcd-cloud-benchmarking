// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.1
// 	protoc        v5.29.3
// source: benchmarkpb/benchmark.proto

package benchmarkpb

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

type CTRLMessage struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Types that are valid to be assigned to Payload:
	//
	//	*CTRLMessage_BenchmarkStatus
	//	*CTRLMessage_ConfigFile
	//	*CTRLMessage_ConfigFileResponse
	//	*CTRLMessage_Shutdown
	//	*CTRLMessage_BenchmarkFinished
	Payload       isCTRLMessage_Payload `protobuf_oneof:"payload"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CTRLMessage) Reset() {
	*x = CTRLMessage{}
	mi := &file_benchmarkpb_benchmark_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CTRLMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CTRLMessage) ProtoMessage() {}

func (x *CTRLMessage) ProtoReflect() protoreflect.Message {
	mi := &file_benchmarkpb_benchmark_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CTRLMessage.ProtoReflect.Descriptor instead.
func (*CTRLMessage) Descriptor() ([]byte, []int) {
	return file_benchmarkpb_benchmark_proto_rawDescGZIP(), []int{0}
}

func (x *CTRLMessage) GetPayload() isCTRLMessage_Payload {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *CTRLMessage) GetBenchmarkStatus() *BenchmarkStatus {
	if x != nil {
		if x, ok := x.Payload.(*CTRLMessage_BenchmarkStatus); ok {
			return x.BenchmarkStatus
		}
	}
	return nil
}

func (x *CTRLMessage) GetConfigFile() *ConfigFile {
	if x != nil {
		if x, ok := x.Payload.(*CTRLMessage_ConfigFile); ok {
			return x.ConfigFile
		}
	}
	return nil
}

func (x *CTRLMessage) GetConfigFileResponse() *ConfigFileResponse {
	if x != nil {
		if x, ok := x.Payload.(*CTRLMessage_ConfigFileResponse); ok {
			return x.ConfigFileResponse
		}
	}
	return nil
}

func (x *CTRLMessage) GetShutdown() *Shutdown {
	if x != nil {
		if x, ok := x.Payload.(*CTRLMessage_Shutdown); ok {
			return x.Shutdown
		}
	}
	return nil
}

func (x *CTRLMessage) GetBenchmarkFinished() *BenchmarkFinished {
	if x != nil {
		if x, ok := x.Payload.(*CTRLMessage_BenchmarkFinished); ok {
			return x.BenchmarkFinished
		}
	}
	return nil
}

type isCTRLMessage_Payload interface {
	isCTRLMessage_Payload()
}

type CTRLMessage_BenchmarkStatus struct {
	BenchmarkStatus *BenchmarkStatus `protobuf:"bytes,1,opt,name=benchmark_status,json=benchmarkStatus,proto3,oneof"`
}

type CTRLMessage_ConfigFile struct {
	ConfigFile *ConfigFile `protobuf:"bytes,3,opt,name=config_file,json=configFile,proto3,oneof"`
}

type CTRLMessage_ConfigFileResponse struct {
	ConfigFileResponse *ConfigFileResponse `protobuf:"bytes,4,opt,name=config_file_response,json=configFileResponse,proto3,oneof"`
}

type CTRLMessage_Shutdown struct {
	Shutdown *Shutdown `protobuf:"bytes,5,opt,name=shutdown,proto3,oneof"`
}

type CTRLMessage_BenchmarkFinished struct {
	BenchmarkFinished *BenchmarkFinished `protobuf:"bytes,6,opt,name=benchmark_finished,json=benchmarkFinished,proto3,oneof"`
}

func (*CTRLMessage_BenchmarkStatus) isCTRLMessage_Payload() {}

func (*CTRLMessage_ConfigFile) isCTRLMessage_Payload() {}

func (*CTRLMessage_ConfigFileResponse) isCTRLMessage_Payload() {}

func (*CTRLMessage_Shutdown) isCTRLMessage_Payload() {}

func (*CTRLMessage_BenchmarkFinished) isCTRLMessage_Payload() {}

type ConfigFile struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Content       []byte                 `protobuf:"bytes,1,opt,name=content,proto3" json:"content,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ConfigFile) Reset() {
	*x = ConfigFile{}
	mi := &file_benchmarkpb_benchmark_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ConfigFile) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConfigFile) ProtoMessage() {}

func (x *ConfigFile) ProtoReflect() protoreflect.Message {
	mi := &file_benchmarkpb_benchmark_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConfigFile.ProtoReflect.Descriptor instead.
func (*ConfigFile) Descriptor() ([]byte, []int) {
	return file_benchmarkpb_benchmark_proto_rawDescGZIP(), []int{1}
}

func (x *ConfigFile) GetContent() []byte {
	if x != nil {
		return x.Content
	}
	return nil
}

type ConfigFileResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ConfigFileResponse) Reset() {
	*x = ConfigFileResponse{}
	mi := &file_benchmarkpb_benchmark_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ConfigFileResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConfigFileResponse) ProtoMessage() {}

func (x *ConfigFileResponse) ProtoReflect() protoreflect.Message {
	mi := &file_benchmarkpb_benchmark_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConfigFileResponse.ProtoReflect.Descriptor instead.
func (*ConfigFileResponse) Descriptor() ([]byte, []int) {
	return file_benchmarkpb_benchmark_proto_rawDescGZIP(), []int{2}
}

func (x *ConfigFileResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

type BenchmarkStatus struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Status        string                 `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BenchmarkStatus) Reset() {
	*x = BenchmarkStatus{}
	mi := &file_benchmarkpb_benchmark_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BenchmarkStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BenchmarkStatus) ProtoMessage() {}

func (x *BenchmarkStatus) ProtoReflect() protoreflect.Message {
	mi := &file_benchmarkpb_benchmark_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BenchmarkStatus.ProtoReflect.Descriptor instead.
func (*BenchmarkStatus) Descriptor() ([]byte, []int) {
	return file_benchmarkpb_benchmark_proto_rawDescGZIP(), []int{3}
}

func (x *BenchmarkStatus) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

type Shutdown struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Shutdown) Reset() {
	*x = Shutdown{}
	mi := &file_benchmarkpb_benchmark_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Shutdown) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Shutdown) ProtoMessage() {}

func (x *Shutdown) ProtoReflect() protoreflect.Message {
	mi := &file_benchmarkpb_benchmark_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Shutdown.ProtoReflect.Descriptor instead.
func (*Shutdown) Descriptor() ([]byte, []int) {
	return file_benchmarkpb_benchmark_proto_rawDescGZIP(), []int{4}
}

type BenchmarkFinished struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BenchmarkFinished) Reset() {
	*x = BenchmarkFinished{}
	mi := &file_benchmarkpb_benchmark_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BenchmarkFinished) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BenchmarkFinished) ProtoMessage() {}

func (x *BenchmarkFinished) ProtoReflect() protoreflect.Message {
	mi := &file_benchmarkpb_benchmark_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BenchmarkFinished.ProtoReflect.Descriptor instead.
func (*BenchmarkFinished) Descriptor() ([]byte, []int) {
	return file_benchmarkpb_benchmark_proto_rawDescGZIP(), []int{5}
}

var File_benchmarkpb_benchmark_proto protoreflect.FileDescriptor

var file_benchmarkpb_benchmark_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x62, 0x65, 0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x70, 0x62, 0x2f, 0x62, 0x65,
	0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x62,
	0x65, 0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x70, 0x62, 0x22, 0xfa, 0x02, 0x0a, 0x0b, 0x43,
	0x54, 0x52, 0x4c, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x49, 0x0a, 0x10, 0x62, 0x65,
	0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x62, 0x65, 0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b,
	0x70, 0x62, 0x2e, 0x42, 0x65, 0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x53, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x48, 0x00, 0x52, 0x0f, 0x62, 0x65, 0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x3a, 0x0a, 0x0b, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f,
	0x66, 0x69, 0x6c, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x62, 0x65, 0x6e,
	0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x70, 0x62, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x46,
	0x69, 0x6c, 0x65, 0x48, 0x00, 0x52, 0x0a, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x46, 0x69, 0x6c,
	0x65, 0x12, 0x53, 0x0a, 0x14, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x66, 0x69, 0x6c, 0x65,
	0x5f, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1f, 0x2e, 0x62, 0x65, 0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x70, 0x62, 0x2e, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x48, 0x00, 0x52, 0x12, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x33, 0x0a, 0x08, 0x73, 0x68, 0x75, 0x74, 0x64, 0x6f,
	0x77, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x62, 0x65, 0x6e, 0x63, 0x68,
	0x6d, 0x61, 0x72, 0x6b, 0x70, 0x62, 0x2e, 0x53, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x48,
	0x00, 0x52, 0x08, 0x73, 0x68, 0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x12, 0x4f, 0x0a, 0x12, 0x62,
	0x65, 0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x5f, 0x66, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x65,
	0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x62, 0x65, 0x6e, 0x63, 0x68, 0x6d,
	0x61, 0x72, 0x6b, 0x70, 0x62, 0x2e, 0x42, 0x65, 0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x46,
	0x69, 0x6e, 0x69, 0x73, 0x68, 0x65, 0x64, 0x48, 0x00, 0x52, 0x11, 0x62, 0x65, 0x6e, 0x63, 0x68,
	0x6d, 0x61, 0x72, 0x6b, 0x46, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x65, 0x64, 0x42, 0x09, 0x0a, 0x07,
	0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x22, 0x26, 0x0a, 0x0a, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x22,
	0x2e, 0x0a, 0x12, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x22,
	0x29, 0x0a, 0x0f, 0x42, 0x65, 0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x53, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x22, 0x0a, 0x0a, 0x08, 0x53, 0x68,
	0x75, 0x74, 0x64, 0x6f, 0x77, 0x6e, 0x22, 0x13, 0x0a, 0x11, 0x42, 0x65, 0x6e, 0x63, 0x68, 0x6d,
	0x61, 0x72, 0x6b, 0x46, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x65, 0x64, 0x32, 0x5a, 0x0a, 0x10, 0x42,
	0x65, 0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12,
	0x46, 0x0a, 0x0a, 0x43, 0x54, 0x52, 0x4c, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12, 0x18, 0x2e,
	0x62, 0x65, 0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x70, 0x62, 0x2e, 0x43, 0x54, 0x52, 0x4c,
	0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x1a, 0x18, 0x2e, 0x62, 0x65, 0x6e, 0x63, 0x68, 0x6d,
	0x61, 0x72, 0x6b, 0x70, 0x62, 0x2e, 0x43, 0x54, 0x52, 0x4c, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x22, 0x00, 0x28, 0x01, 0x30, 0x01, 0x42, 0x15, 0x5a, 0x13, 0x63, 0x73, 0x62, 0x2f, 0x61,
	0x70, 0x69, 0x2f, 0x62, 0x65, 0x6e, 0x63, 0x68, 0x6d, 0x61, 0x72, 0x6b, 0x70, 0x62, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_benchmarkpb_benchmark_proto_rawDescOnce sync.Once
	file_benchmarkpb_benchmark_proto_rawDescData = file_benchmarkpb_benchmark_proto_rawDesc
)

func file_benchmarkpb_benchmark_proto_rawDescGZIP() []byte {
	file_benchmarkpb_benchmark_proto_rawDescOnce.Do(func() {
		file_benchmarkpb_benchmark_proto_rawDescData = protoimpl.X.CompressGZIP(file_benchmarkpb_benchmark_proto_rawDescData)
	})
	return file_benchmarkpb_benchmark_proto_rawDescData
}

var file_benchmarkpb_benchmark_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_benchmarkpb_benchmark_proto_goTypes = []any{
	(*CTRLMessage)(nil),        // 0: benchmarkpb.CTRLMessage
	(*ConfigFile)(nil),         // 1: benchmarkpb.ConfigFile
	(*ConfigFileResponse)(nil), // 2: benchmarkpb.ConfigFileResponse
	(*BenchmarkStatus)(nil),    // 3: benchmarkpb.BenchmarkStatus
	(*Shutdown)(nil),           // 4: benchmarkpb.Shutdown
	(*BenchmarkFinished)(nil),  // 5: benchmarkpb.BenchmarkFinished
}
var file_benchmarkpb_benchmark_proto_depIdxs = []int32{
	3, // 0: benchmarkpb.CTRLMessage.benchmark_status:type_name -> benchmarkpb.BenchmarkStatus
	1, // 1: benchmarkpb.CTRLMessage.config_file:type_name -> benchmarkpb.ConfigFile
	2, // 2: benchmarkpb.CTRLMessage.config_file_response:type_name -> benchmarkpb.ConfigFileResponse
	4, // 3: benchmarkpb.CTRLMessage.shutdown:type_name -> benchmarkpb.Shutdown
	5, // 4: benchmarkpb.CTRLMessage.benchmark_finished:type_name -> benchmarkpb.BenchmarkFinished
	0, // 5: benchmarkpb.BenchmarkService.CTRLStream:input_type -> benchmarkpb.CTRLMessage
	0, // 6: benchmarkpb.BenchmarkService.CTRLStream:output_type -> benchmarkpb.CTRLMessage
	6, // [6:7] is the sub-list for method output_type
	5, // [5:6] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_benchmarkpb_benchmark_proto_init() }
func file_benchmarkpb_benchmark_proto_init() {
	if File_benchmarkpb_benchmark_proto != nil {
		return
	}
	file_benchmarkpb_benchmark_proto_msgTypes[0].OneofWrappers = []any{
		(*CTRLMessage_BenchmarkStatus)(nil),
		(*CTRLMessage_ConfigFile)(nil),
		(*CTRLMessage_ConfigFileResponse)(nil),
		(*CTRLMessage_Shutdown)(nil),
		(*CTRLMessage_BenchmarkFinished)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_benchmarkpb_benchmark_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_benchmarkpb_benchmark_proto_goTypes,
		DependencyIndexes: file_benchmarkpb_benchmark_proto_depIdxs,
		MessageInfos:      file_benchmarkpb_benchmark_proto_msgTypes,
	}.Build()
	File_benchmarkpb_benchmark_proto = out.File
	file_benchmarkpb_benchmark_proto_rawDesc = nil
	file_benchmarkpb_benchmark_proto_goTypes = nil
	file_benchmarkpb_benchmark_proto_depIdxs = nil
}
