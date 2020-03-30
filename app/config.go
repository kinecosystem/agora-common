package app

import (
	"time"
)

// AppConfig is the application specific configuration.
// It is passed to the App.Init function, and is optional.
type AppConfig map[string]interface{}

// Config contains the base configuration for agora services, as well as the
// application itself.
type Config struct {
	LogLevel string `mapstructure:"log_level"`
	LogType  string `mapstructure:"log_type"`

	ListenAddress       string        `mapstructure:"listen_address"`
	ShutdownGracePeriod time.Duration `mapstructure:"shutdown_grace_period"`

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
	AppConfig AppConfig `mapstructure:"app"`
}

var defaultConfig = Config{
	LogType: "json",

	ListenAddress:       ":8085",
	ShutdownGracePeriod: 30 * time.Second,

	EnablePprof:        true,
	EnableExpvar:       true,
	DebugListenAddress: ":8123",
}
