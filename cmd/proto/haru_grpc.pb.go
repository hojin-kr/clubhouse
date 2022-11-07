// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.19.4
// source: proto/haru.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// Version1Client is the client API for Version1 service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type Version1Client interface {
	CreateAccount(ctx context.Context, in *AccountRequest, opts ...grpc.CallOption) (*AccountReply, error)
	GetProfile(ctx context.Context, in *ProfileRequest, opts ...grpc.CallOption) (*ProfileReply, error)
	UpdateProfile(ctx context.Context, in *ProfileRequest, opts ...grpc.CallOption) (*ProfileReply, error)
	CreateRound(ctx context.Context, in *RoundRequest, opts ...grpc.CallOption) (*RoundReply, error)
	UpdateRound(ctx context.Context, in *RoundRequest, opts ...grpc.CallOption) (*RoundReply, error)
	GetRound(ctx context.Context, in *RoundRequest, opts ...grpc.CallOption) (*RoundReply, error)
}

type version1Client struct {
	cc grpc.ClientConnInterface
}

func NewVersion1Client(cc grpc.ClientConnInterface) Version1Client {
	return &version1Client{cc}
}

func (c *version1Client) CreateAccount(ctx context.Context, in *AccountRequest, opts ...grpc.CallOption) (*AccountReply, error) {
	out := new(AccountReply)
	err := c.cc.Invoke(ctx, "/haru.version1/CreateAccount", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *version1Client) GetProfile(ctx context.Context, in *ProfileRequest, opts ...grpc.CallOption) (*ProfileReply, error) {
	out := new(ProfileReply)
	err := c.cc.Invoke(ctx, "/haru.version1/GetProfile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *version1Client) UpdateProfile(ctx context.Context, in *ProfileRequest, opts ...grpc.CallOption) (*ProfileReply, error) {
	out := new(ProfileReply)
	err := c.cc.Invoke(ctx, "/haru.version1/UpdateProfile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *version1Client) CreateRound(ctx context.Context, in *RoundRequest, opts ...grpc.CallOption) (*RoundReply, error) {
	out := new(RoundReply)
	err := c.cc.Invoke(ctx, "/haru.version1/CreateRound", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *version1Client) UpdateRound(ctx context.Context, in *RoundRequest, opts ...grpc.CallOption) (*RoundReply, error) {
	out := new(RoundReply)
	err := c.cc.Invoke(ctx, "/haru.version1/UpdateRound", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *version1Client) GetRound(ctx context.Context, in *RoundRequest, opts ...grpc.CallOption) (*RoundReply, error) {
	out := new(RoundReply)
	err := c.cc.Invoke(ctx, "/haru.version1/GetRound", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Version1Server is the server API for Version1 service.
// All implementations must embed UnimplementedVersion1Server
// for forward compatibility
type Version1Server interface {
	CreateAccount(context.Context, *AccountRequest) (*AccountReply, error)
	GetProfile(context.Context, *ProfileRequest) (*ProfileReply, error)
	UpdateProfile(context.Context, *ProfileRequest) (*ProfileReply, error)
	CreateRound(context.Context, *RoundRequest) (*RoundReply, error)
	UpdateRound(context.Context, *RoundRequest) (*RoundReply, error)
	GetRound(context.Context, *RoundRequest) (*RoundReply, error)
	mustEmbedUnimplementedVersion1Server()
}

// UnimplementedVersion1Server must be embedded to have forward compatible implementations.
type UnimplementedVersion1Server struct {
}

func (UnimplementedVersion1Server) CreateAccount(context.Context, *AccountRequest) (*AccountReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateAccount not implemented")
}
func (UnimplementedVersion1Server) GetProfile(context.Context, *ProfileRequest) (*ProfileReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetProfile not implemented")
}
func (UnimplementedVersion1Server) UpdateProfile(context.Context, *ProfileRequest) (*ProfileReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateProfile not implemented")
}
func (UnimplementedVersion1Server) CreateRound(context.Context, *RoundRequest) (*RoundReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateRound not implemented")
}
func (UnimplementedVersion1Server) UpdateRound(context.Context, *RoundRequest) (*RoundReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateRound not implemented")
}
func (UnimplementedVersion1Server) GetRound(context.Context, *RoundRequest) (*RoundReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRound not implemented")
}
func (UnimplementedVersion1Server) mustEmbedUnimplementedVersion1Server() {}

// UnsafeVersion1Server may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to Version1Server will
// result in compilation errors.
type UnsafeVersion1Server interface {
	mustEmbedUnimplementedVersion1Server()
}

func RegisterVersion1Server(s grpc.ServiceRegistrar, srv Version1Server) {
	s.RegisterService(&Version1_ServiceDesc, srv)
}

func _Version1_CreateAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AccountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Version1Server).CreateAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/haru.version1/CreateAccount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(Version1Server).CreateAccount(ctx, req.(*AccountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Version1_GetProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProfileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Version1Server).GetProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/haru.version1/GetProfile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(Version1Server).GetProfile(ctx, req.(*ProfileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Version1_UpdateProfile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProfileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Version1Server).UpdateProfile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/haru.version1/UpdateProfile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(Version1Server).UpdateProfile(ctx, req.(*ProfileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Version1_CreateRound_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoundRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Version1Server).CreateRound(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/haru.version1/CreateRound",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(Version1Server).CreateRound(ctx, req.(*RoundRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Version1_UpdateRound_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoundRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Version1Server).UpdateRound(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/haru.version1/UpdateRound",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(Version1Server).UpdateRound(ctx, req.(*RoundRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Version1_GetRound_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoundRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Version1Server).GetRound(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/haru.version1/GetRound",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(Version1Server).GetRound(ctx, req.(*RoundRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Version1_ServiceDesc is the grpc.ServiceDesc for Version1 service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Version1_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "haru.version1",
	HandlerType: (*Version1Server)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateAccount",
			Handler:    _Version1_CreateAccount_Handler,
		},
		{
			MethodName: "GetProfile",
			Handler:    _Version1_GetProfile_Handler,
		},
		{
			MethodName: "UpdateProfile",
			Handler:    _Version1_UpdateProfile_Handler,
		},
		{
			MethodName: "CreateRound",
			Handler:    _Version1_CreateRound_Handler,
		},
		{
			MethodName: "UpdateRound",
			Handler:    _Version1_UpdateRound_Handler,
		},
		{
			MethodName: "GetRound",
			Handler:    _Version1_GetRound_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/haru.proto",
}
