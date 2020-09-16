package app

import (
	"crypto/tls"
	"expvar"
	"flag"
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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/kinecosystem/agora-common/headers"
	"github.com/kinecosystem/agora-common/protobuf/validation"
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
	Init(config Config) error

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
			headers.UnaryServerInterceptor(),
		},
		streamServerInterceptors: []grpc.StreamServerInterceptor{
			validation.StreamServerInterceptor(),
			headers.StreamServerInterceptor(),
		},
	}
	for _, o := range options {
		o(&opts)
	}

	_ = viper.BindEnv("listen_address", "LISTEN_ADDRESS")
	_ = viper.BindEnv("debug_listen_address", "DEBUG_LISTEN_ADDRESS")
	_ = viper.BindEnv("log_level", "LOG_LEVEL")
	_ = viper.BindEnv("log_type", "LOG_TYPE")
	_ = viper.BindEnv("tls_certificate", "TLS_CERTIFICATE")
	_ = viper.BindEnv("tls_private_key", "TLS_PRIVATE_KEY")

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

	var transportCreds credentials.TransportCredentials

	if config.TLSCertificate != "" {
		if config.TLSKey == "" {
			logger.Error("tls key must be provided if certificate is specified")
			os.Exit(1)
		}

		certBytes, err := LoadFile(config.TLSCertificate)
		if err != nil {
			logger.WithError(err).Error("failed to load tls certificate")
			os.Exit(1)
		}

		keyBytes, err := LoadFile(config.TLSKey)
		if err != nil {
			logger.WithError(err).Error("failed to load tls key")
			os.Exit(1)
		}

		cert, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			logger.WithError(err).Error("invalid certificate/private key")
			os.Exit(1)
		}

		transportCreds = credentials.NewServerTLSFromCert(&cert)
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
		grpc.Creds(transportCreds),
		grpc_middleware.WithUnaryServerChain(
			append([]grpc.UnaryServerInterceptor{grpc_prometheus.UnaryServerInterceptor}, opts.unaryServerInterceptors...)...,
		),
		grpc_middleware.WithStreamServerChain(
			append([]grpc.StreamServerInterceptor{grpc_prometheus.StreamServerInterceptor}, opts.streamServerInterceptors...)...,
		),
	)

	app.RegisterWithGRPC(serv)
	grpc_prometheus.Register(serv)
	grpc_prometheus.EnableHandlingTimeHistogram()
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

type prometheusLogger struct {
	warnCounter  prometheus.Counter
	errorCounter prometheus.Counter
}

func newPrometheusLogger() *prometheusLogger {
	l := &prometheusLogger{}
	l.warnCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "logging_warns",
		Namespace: "agora",
	})
	if err := prometheus.Register(l.warnCounter); err != nil {
		if e, ok := err.(prometheus.AlreadyRegisteredError); ok {
			l.warnCounter = e.ExistingCollector.(prometheus.Counter)
		}
	}

	l.errorCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "logging_errors",
		Namespace: "agora",
	})
	if err := prometheus.Register(l.errorCounter); err != nil {
		if e, ok := err.(prometheus.AlreadyRegisteredError); ok {
			l.errorCounter = e.ExistingCollector.(prometheus.Counter)
		}
	}

	return l
}

func (p *prometheusLogger) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.WarnLevel,
		logrus.ErrorLevel,
	}
}

func (p *prometheusLogger) Fire(e *logrus.Entry) error {
	switch e.Level {
	case logrus.WarnLevel:
		p.warnCounter.Inc()
	case logrus.ErrorLevel:
		p.errorCounter.Inc()
	}

	return nil
}

func configureLogger(config BaseConfig) {
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
	logrus.StandardLogger().Hooks.Add(newPrometheusLogger())
}
