// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v4.25.1
// source: pb/remote_exec.proto

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	RemoteExecutor_ExecCommand_FullMethodName = "/RemoteExecutor/ExecCommand"
	RemoteExecutor_GetStatus_FullMethodName   = "/RemoteExecutor/GetStatus"
	RemoteExecutor_GetOutput_FullMethodName   = "/RemoteExecutor/GetOutput"
	RemoteExecutor_StopJob_FullMethodName     = "/RemoteExecutor/StopJob"
)

// RemoteExecutorClient is the client API for RemoteExecutor service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RemoteExecutorClient interface {
	// Runs a command on the server and returns the Job details
	ExecCommand(ctx context.Context, in *CmdRequest, opts ...grpc.CallOption) (*JobDetails, error)
	// Gets the status of a Job
	GetStatus(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*JobDetails, error)
	// Gets the output for the requested Job Id
	GetOutput(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[JobOutput], error)
	// Stops a job.
	StopJob(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type remoteExecutorClient struct {
	cc grpc.ClientConnInterface
}

func NewRemoteExecutorClient(cc grpc.ClientConnInterface) RemoteExecutorClient {
	return &remoteExecutorClient{cc}
}

func (c *remoteExecutorClient) ExecCommand(ctx context.Context, in *CmdRequest, opts ...grpc.CallOption) (*JobDetails, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(JobDetails)
	err := c.cc.Invoke(ctx, RemoteExecutor_ExecCommand_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteExecutorClient) GetStatus(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*JobDetails, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(JobDetails)
	err := c.cc.Invoke(ctx, RemoteExecutor_GetStatus_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *remoteExecutorClient) GetOutput(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[JobOutput], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &RemoteExecutor_ServiceDesc.Streams[0], RemoteExecutor_GetOutput_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[GetRequest, JobOutput]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type RemoteExecutor_GetOutputClient = grpc.ServerStreamingClient[JobOutput]

func (c *remoteExecutorClient) StopJob(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, RemoteExecutor_StopJob_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RemoteExecutorServer is the server API for RemoteExecutor service.
// All implementations must embed UnimplementedRemoteExecutorServer
// for forward compatibility.
type RemoteExecutorServer interface {
	// Runs a command on the server and returns the Job details
	ExecCommand(context.Context, *CmdRequest) (*JobDetails, error)
	// Gets the status of a Job
	GetStatus(context.Context, *GetRequest) (*JobDetails, error)
	// Gets the output for the requested Job Id
	GetOutput(*GetRequest, grpc.ServerStreamingServer[JobOutput]) error
	// Stops a job.
	StopJob(context.Context, *StopRequest) (*emptypb.Empty, error)
	mustEmbedUnimplementedRemoteExecutorServer()
}

// UnimplementedRemoteExecutorServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedRemoteExecutorServer struct{}

func (UnimplementedRemoteExecutorServer) ExecCommand(context.Context, *CmdRequest) (*JobDetails, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ExecCommand not implemented")
}
func (UnimplementedRemoteExecutorServer) GetStatus(context.Context, *GetRequest) (*JobDetails, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStatus not implemented")
}
func (UnimplementedRemoteExecutorServer) GetOutput(*GetRequest, grpc.ServerStreamingServer[JobOutput]) error {
	return status.Errorf(codes.Unimplemented, "method GetOutput not implemented")
}
func (UnimplementedRemoteExecutorServer) StopJob(context.Context, *StopRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StopJob not implemented")
}
func (UnimplementedRemoteExecutorServer) mustEmbedUnimplementedRemoteExecutorServer() {}
func (UnimplementedRemoteExecutorServer) testEmbeddedByValue()                        {}

// UnsafeRemoteExecutorServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RemoteExecutorServer will
// result in compilation errors.
type UnsafeRemoteExecutorServer interface {
	mustEmbedUnimplementedRemoteExecutorServer()
}

func RegisterRemoteExecutorServer(s grpc.ServiceRegistrar, srv RemoteExecutorServer) {
	// If the following call pancis, it indicates UnimplementedRemoteExecutorServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&RemoteExecutor_ServiceDesc, srv)
}

func _RemoteExecutor_ExecCommand_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CmdRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteExecutorServer).ExecCommand(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RemoteExecutor_ExecCommand_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteExecutorServer).ExecCommand(ctx, req.(*CmdRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteExecutor_GetStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteExecutorServer).GetStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RemoteExecutor_GetStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteExecutorServer).GetStatus(ctx, req.(*GetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RemoteExecutor_GetOutput_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(RemoteExecutorServer).GetOutput(m, &grpc.GenericServerStream[GetRequest, JobOutput]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type RemoteExecutor_GetOutputServer = grpc.ServerStreamingServer[JobOutput]

func _RemoteExecutor_StopJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StopRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RemoteExecutorServer).StopJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RemoteExecutor_StopJob_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RemoteExecutorServer).StopJob(ctx, req.(*StopRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RemoteExecutor_ServiceDesc is the grpc.ServiceDesc for RemoteExecutor service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RemoteExecutor_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "RemoteExecutor",
	HandlerType: (*RemoteExecutorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ExecCommand",
			Handler:    _RemoteExecutor_ExecCommand_Handler,
		},
		{
			MethodName: "GetStatus",
			Handler:    _RemoteExecutor_GetStatus_Handler,
		},
		{
			MethodName: "StopJob",
			Handler:    _RemoteExecutor_StopJob_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetOutput",
			Handler:       _RemoteExecutor_GetOutput_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "pb/remote_exec.proto",
}
