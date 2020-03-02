package testutil

import (
	"fmt"
	"github.com/kinecosystem/agora-common/netutil"
	"github.com/kinecosystem/agora-common/protobuf/validation"
	"net"
	"sync"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Server provides a local gRPC server with basic agora interceptors that
// can be used for testing with no external dependencies.
type Server struct {
	closeFunc sync.Once

	sync.Mutex
	serv       bool
	listener   net.Listener
	grpcServer *grpc.Server
}

// NewServer creates a new Server.
func NewServer(opts ...ServerOption) (*grpc.ClientConn, *Server, error) {
	port, err := netutil.GetAvailablePortForAddress("localhost")
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to find free port")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to start listener")
	}

	o := serverOpts{
		unaryClientInterceptors: []grpc.UnaryClientInterceptor{
			validation.UnaryClientInterceptor(),
		},
		streamClientInterceptors: []grpc.StreamClientInterceptor{
			validation.StreamClientInterceptor(),
		},
		unaryServerInterceptors: []grpc.UnaryServerInterceptor{
			validation.UnaryServerInterceptor(),
		},
		streamServerInterceptors: []grpc.StreamServerInterceptor{
			validation.StreamServerInterceptor(),
		},
	}

	for _, opt := range opts {
		opt(&o)
	}

	// note: this is safe since we don't specify grpc.WithBlock()
	conn, err := grpc.Dial(
		fmt.Sprintf("localhost:%d", port),
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(o.unaryClientInterceptors...)),
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(o.streamClientInterceptors...)),
	)

	if err != nil {
		listener.Close()
		return nil, nil, errors.Wrapf(err, "failed to create grpc.ClientConn")
	}

	return conn, &Server{
		listener: listener,
		grpcServer: grpc.NewServer(
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(o.unaryServerInterceptors...)),
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(o.streamServerInterceptors...)),
		),
	}, nil
}

// RegisterService registers a gRPC service with Server.
func (s *Server) RegisterService(registerFunc func(s *grpc.Server)) {
	registerFunc(s.grpcServer)
}

// Serve asynchronously starts the server, provided it has not been previously
// started or stopped. Callers should use stopFunc to stop the server in order
// to cleanup the underlying resources.
func (s *Server) Serve() (stopFunc func(), err error) {
	s.Lock()
	defer s.Unlock()

	if s.serv {
		return
	}

	if s.grpcServer == nil {
		return nil, errors.Errorf("testserver already stopped")
	}

	stopFunc = func() {
		s.closeFunc.Do(func() {
			s.Lock()
			defer s.Unlock()

			s.grpcServer.Stop()
			s.listener.Close()
			s.grpcServer = nil
			s.listener = nil
		})
	}

	go func() {
		s.Lock()
		lis := s.listener
		serv := s.grpcServer
		s.Unlock()

		if lis == nil || serv == nil {
			return
		}

		err := serv.Serve(lis)
		logrus.
			StandardLogger().
			WithField("type", "testutil/server").
			WithError(err).
			Debug("stopped")
		stopFunc()
	}()

	s.serv = true
	return stopFunc, nil
}
