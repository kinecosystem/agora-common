package testutil

import "google.golang.org/grpc"

type serverOpts struct {
	unaryClientInterceptors  []grpc.UnaryClientInterceptor
	streamClientInterceptors []grpc.StreamClientInterceptor

	unaryServerInterceptors  []grpc.UnaryServerInterceptor
	streamServerInterceptors []grpc.StreamServerInterceptor
}

// ServerOption configures the settings when creating a test server.
type ServerOption func(o *serverOpts)

// WithUnaryClientInterceptor adds a unary client interceptor to the test client.
func WithUnaryClientInterceptor(i grpc.UnaryClientInterceptor) ServerOption {
	return func(o *serverOpts) {
		o.unaryClientInterceptors = append(o.unaryClientInterceptors, i)
	}
}

// WithStreamClientInterceptor adds a stream client interceptor to the test client.
func WithStreamClientInterceptor(i grpc.StreamClientInterceptor) ServerOption {
	return func(o *serverOpts) {
		o.streamClientInterceptors = append(o.streamClientInterceptors, i)
	}
}

// WithUnaryServerInterceptor adds a unary server interceptor to the test client.
func WithUnaryServerInterceptor(i grpc.UnaryServerInterceptor) ServerOption {
	return func(o *serverOpts) {
		o.unaryServerInterceptors = append(o.unaryServerInterceptors, i)
	}
}

// WithStreamServerInterceptor adds a stream server interceptor to the test client.
func WithStreamServerInterceptor(i grpc.StreamServerInterceptor) ServerOption {
	return func(o *serverOpts) {
		o.streamServerInterceptors = append(o.streamServerInterceptors, i)
	}
}
