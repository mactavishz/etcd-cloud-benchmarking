// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: benchmarkpb/benchmark.proto

package benchmarkpb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	BenchmarkService_CTRLStream_FullMethodName = "/benchmarkpb.BenchmarkService/CTRLStream"
)

// BenchmarkServiceClient is the client API for BenchmarkService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// BenchmarkService defines the interface between control and client
type BenchmarkServiceClient interface {
	CTRLStream(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[CTRLMessage, CTRLMessage], error)
}

type benchmarkServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBenchmarkServiceClient(cc grpc.ClientConnInterface) BenchmarkServiceClient {
	return &benchmarkServiceClient{cc}
}

func (c *benchmarkServiceClient) CTRLStream(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[CTRLMessage, CTRLMessage], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &BenchmarkService_ServiceDesc.Streams[0], BenchmarkService_CTRLStream_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[CTRLMessage, CTRLMessage]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type BenchmarkService_CTRLStreamClient = grpc.BidiStreamingClient[CTRLMessage, CTRLMessage]

// BenchmarkServiceServer is the server API for BenchmarkService service.
// All implementations must embed UnimplementedBenchmarkServiceServer
// for forward compatibility.
//
// BenchmarkService defines the interface between control and client
type BenchmarkServiceServer interface {
	CTRLStream(grpc.BidiStreamingServer[CTRLMessage, CTRLMessage]) error
	mustEmbedUnimplementedBenchmarkServiceServer()
}

// UnimplementedBenchmarkServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedBenchmarkServiceServer struct{}

func (UnimplementedBenchmarkServiceServer) CTRLStream(grpc.BidiStreamingServer[CTRLMessage, CTRLMessage]) error {
	return status.Errorf(codes.Unimplemented, "method CTRLStream not implemented")
}
func (UnimplementedBenchmarkServiceServer) mustEmbedUnimplementedBenchmarkServiceServer() {}
func (UnimplementedBenchmarkServiceServer) testEmbeddedByValue()                          {}

// UnsafeBenchmarkServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BenchmarkServiceServer will
// result in compilation errors.
type UnsafeBenchmarkServiceServer interface {
	mustEmbedUnimplementedBenchmarkServiceServer()
}

func RegisterBenchmarkServiceServer(s grpc.ServiceRegistrar, srv BenchmarkServiceServer) {
	// If the following call pancis, it indicates UnimplementedBenchmarkServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&BenchmarkService_ServiceDesc, srv)
}

func _BenchmarkService_CTRLStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(BenchmarkServiceServer).CTRLStream(&grpc.GenericServerStream[CTRLMessage, CTRLMessage]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type BenchmarkService_CTRLStreamServer = grpc.BidiStreamingServer[CTRLMessage, CTRLMessage]

// BenchmarkService_ServiceDesc is the grpc.ServiceDesc for BenchmarkService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BenchmarkService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "benchmarkpb.BenchmarkService",
	HandlerType: (*BenchmarkServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "CTRLStream",
			Handler:       _BenchmarkService_CTRLStream_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "benchmarkpb/benchmark.proto",
}
