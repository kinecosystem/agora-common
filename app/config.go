package app

import (
	"time"
)

// Config is the application specific configuration.
// It is passed to the App.Init function, and is optional.
type Config map[string]interface{}

// BaseConfig contains the base configuration for agora services, as well as the
// application itself.
type BaseConfig struct {
	LogLevel string `mapstructure:"log_level"`
	LogType  string `mapstructure:"log_type"`

	ListenAddress         string        `mapstructure:"listen_address"`
	InsecureListenAddress string        `mapstructure:"insecure_listen_address"`
	ShutdownGracePeriod   time.Duration `mapstructure:"shutdown_grace_period"`

	HTTPGatewayAddress string `mapstructure:"http_gateway_address"`

	// TLSCertificate is an optional URL that specified a TLS certificate to be
	// used for the gRPC server.
	//
	// Currently only two supported URL schemes are supported: file, s3.
	// If no scheme is specified, file is used.
	TLSCertificate string `mapstructure:"tls_certificate"`
	// TLSKey is an optional URL that specifies a TLS Private Key to be used for the
	// gRPC server.
	//
	// Currently only two supported URL schemes are supported: file, s3.
	// If no scheme is specified, file is used.
	TLSKey string `mapstructure:"tls_private_key"`

	EnablePprof        bool   `mapstructure:"enable_pprof"`
	EnableExpvar       bool   `mapstructure:"enable_expvar"`
	DebugListenAddress string `mapstructure:"debug_listen_address"`

	// Arbitrary configuration that the service can define / implement.
	//
	// Users should use mapstructure.Decode for ServiceConfig.
	AppConfig Config `mapstructure:"app"`
}

var defaultConfig = BaseConfig{
	LogType: "json",

	ListenAddress:         ":8085",
	InsecureListenAddress: "localhost:8086",
	ShutdownGracePeriod:   30 * time.Second,

	HTTPGatewayAddress: ":8080",

	EnablePprof:        true,
	EnableExpvar:       true,
	DebugListenAddress: ":8123",
}
