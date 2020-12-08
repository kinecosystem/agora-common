package app

import (
	"google.golang.org/grpc"

	"github.com/kinecosystem/agora-common/httpgateway"
)

// Option configures the environment run by Run().
type Option func(o *opts)

type opts struct {
	unaryServerInterceptors  []grpc.UnaryServerInterceptor
	streamServerInterceptors []grpc.StreamServerInterceptor

	httpGatewayEnabled bool
	httpGatewayOptions []httpgateway.MuxOption
}

// WithUnaryServerInterceptor configures the app's gRPC server to use the provided interceptor.
//
// Interceptors are evaluated in addition order, and configured interceptors are executed after
// the app's default interceptors.
func WithUnaryServerInterceptor(interceptor grpc.UnaryServerInterceptor) Option {
	return func(o *opts) {
		o.unaryServerInterceptors = append(o.unaryServerInterceptors, interceptor)
	}
}

// WithStreamServerInterceptor configures the app's gRPC server to use the provided interceptor.
//
// Interceptors are evaluated in addition order, and configured interceptors are executed after
// the app's default interceptors.
func WithStreamServerInterceptor(interceptor grpc.StreamServerInterceptor) Option {
	return func(o *opts) {
		o.streamServerInterceptors = append(o.streamServerInterceptors, interceptor)
	}
}

// WithHTTPGatewayEnabled configures whether or not an HTTP gateway should be enabled with the provided options.
func WithHTTPGatewayEnabled(enabled bool, muxOpts ...httpgateway.MuxOption) Option {
	return func(o *opts) {
		o.httpGatewayEnabled = enabled
		o.httpGatewayOptions = muxOpts
	}
}
