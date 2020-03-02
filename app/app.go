package app

import (
	"expvar"
	"flag"
	"github.com/kinecosystem/agora-common/protobuf/validation"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

// App is a long lived application that services network requests.
// It is expected that App's have gRPC services, but is not a hard requirement.
//
// The lifecycle of the App is tied to the process. The app gets initialized
// before the gRPC server runs, and gets stopped after the gRPC server has stopped
// serving.
type App interface {
	// Init initializes the application in a blocking fashion. When Init returns, it
	// is expected that the application is ready to start receiving requests (provided
	// there are gRPC handlers installed).
	Init(config AppConfig) error

	// RegisterWithGRPC provides a mechanism for the application to register gRPC services
	// with the gRPC server.
	RegisterWithGRPC(server *grpc.Server)

	// ShutdownChan returns a channel that is closed when the application is shutdown.
	//
	// If the channel is closed, the gRPC server will initiate a shutdown if it has
	// not already done so.
	ShutdownChan() <-chan struct{}

	// Stop stops the service, allowing for it to clean up any resources. When Stop()
	// returns, the process exits.
	//
	// Stop should be idempotent.
	Stop()
}

var (
	configPath = flag.String("config", "config.yaml", "configuration file path")

	loadedConfig *Config

	osSigCh = make(chan os.Signal, 1)
)

func init() {
	signal.Notify(osSigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
}

func Run(app App, options ...Option) error {
	flag.Parse()

	opts := opts{
		unaryServerInterceptors: []grpc.UnaryServerInterceptor{
			validation.UnaryServerInterceptor(),
		},
		streamServerInterceptors: []grpc.StreamServerInterceptor{
			validation.StreamServerInterceptor(),
		},
	}
	for _, o := range options {
		o(&opts)
	}

	viper.BindEnv("listen_address", "LISTEN_ADDRESS")
	viper.BindEnv("debug_listen_address", "DEBUG_LISTEN_ADDRESS")
	viper.BindEnv("log_level", "LOG_LEVEL")
	viper.BindEnv("log_type", "LOG_TYPE")

	logger := logrus.StandardLogger().WithField("type", "agora/app")

	// viper.ReadInConfig only returns ConfigFileNotFoundError if it has to search
	// for a default config file because one hasn't been explicitly set. That is,
	// if we explicitly set a config file, and it does not exist, viper will not
	// return a ConfigFileNotFoundError, so we do it ourselves.
	if _, err := os.Stat(*configPath); err == nil {
		viper.SetConfigFile(*configPath)
	} else if !os.IsNotExist(err) {
		logger.WithError(err).Errorf("failed to check if config exists")
		os.Exit(1)
	}

	err := viper.ReadInConfig()
	_, isConfigNotFound := err.(viper.ConfigFileNotFoundError)
	if err != nil && !isConfigNotFound {
		logger.WithError(err).Error("failed to load config")
		os.Exit(1)
	}

	config := defaultConfig
	if err := viper.Unmarshal(&config); err != nil {
		logger.WithError(err).Error("failed to unmarshal config")
		os.Exit(1)
	}

	loadedConfig = &config

	configureLogger(config)

	// We don't want to expose pprof/expvar publically, so we reset the default
	// http ServeMux, which will have those installed due to the init() function
	// in those packages. We expect clients to set up their own HTTP handlers in
	// the Init() func, which is called after this, so this is ok.
	http.DefaultServeMux = http.NewServeMux()

	debugHTTPMux := http.NewServeMux()
	if config.EnableExpvar {
		debugHTTPMux.Handle("/debug/vars", expvar.Handler())
	}
	if config.EnablePprof {
		debugHTTPMux.HandleFunc("/debug/pprof/", pprof.Index)
		debugHTTPMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		debugHTTPMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		debugHTTPMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		debugHTTPMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	if config.EnableExpvar || config.EnablePprof {
		go func() {
			for {
				if err := http.ListenAndServe(config.DebugListenAddress, debugHTTPMux); err != nil {
					logger.WithError(err).Warn("Debug HTTP server failed. Retrying in 5s...")
				}
				time.Sleep(5 * time.Second)
			}
		}()
	}

	// todo: metrics, interceptors, etc

	if err := app.Init(config.AppConfig); err != nil {
		logger.WithError(err).Error("failed to initialize application")
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", config.ListenAddress)
	if err != nil {
		logger.WithError(err).Errorf("failed to listen on %s", config.ListenAddress)
	}

	serv := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			append([]grpc.UnaryServerInterceptor{grpc_prometheus.UnaryServerInterceptor}, opts.unaryServerInterceptors...)...,
		),
		grpc_middleware.WithStreamServerChain(
			append([]grpc.StreamServerInterceptor{grpc_prometheus.StreamServerInterceptor}, opts.streamServerInterceptors...)...,
		),
	)

	app.RegisterWithGRPC(serv)
	grpc_prometheus.Register(serv)
	debugHTTPMux.Handle("/metrics", promhttp.Handler())

	healthgrpc.RegisterHealthServer(serv, health.NewServer())

	servShutdownCh := make(chan struct{})

	go func() {
		if err := serv.Serve(lis); err != nil {
			logger.WithError(err).Error("grpc serve stopped")
		} else {
			logger.Info("grpc server stopped")
		}

		close(servShutdownCh)
	}()

	// Wait for the following shutdown conditions:
	//    1. OS Signal telling us to shutdown
	//    2. The gRPC Server has shutdown (for whatever reason)
	//    3. The application has shutdown (for whatever reason)
	select {
	case <-osSigCh:
		logger.Info("interrupt received, shutting down")
	case <-servShutdownCh:
		logger.Info("grpc server shutdown")
	case <-app.ShutdownChan():
		logger.Info("app shutdown")
	}

	shutdownCh := make(chan struct{})
	go func() {
		// Both the gRPC server and the application should have idempotent
		// shutdown methods, so it's fine call them both, regardless of the
		// shutdown condition.
		serv.GracefulStop()
		app.Stop()

		close(shutdownCh)
	}()

	select {
	case <-shutdownCh:
		return nil
	case <-time.After(config.ShutdownGracePeriod):
		return errors.Errorf("failed to stop the application within %v", config.ShutdownGracePeriod)
	}
}

func configureLogger(config Config) {
	switch strings.ToLower(config.LogType) {
	case "human":
		// The default formatter for logrus is 'human' readable.
	case "", "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		logrus.SetFormatter(&logrus.JSONFormatter{})
		logrus.StandardLogger().WithField("log_type", config.LogType).Warn("unknown logger type, ignoring")
	}

	level, err := logrus.ParseLevel(strings.ToLower(config.LogLevel))
	if err != nil {
		logrus.StandardLogger().WithField("log_level", config.LogLevel).Warn("unknown log level, ignoring")
	} else {
		logrus.SetLevel(level)
	}

	logrus.SetOutput(os.Stdout)
}
