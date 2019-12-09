package env

import (
	"errors"
	"os"
)

// AgoraEnvironment is used to determine which environment the application is currently running in
type AgoraEnvironment string

const (
	// AgoraEnvironmentProd is the production environment
	AgoraEnvironmentProd AgoraEnvironment = "prod"

	// AgoraEnvironmentDev is the development environment
	AgoraEnvironmentDev AgoraEnvironment = "dev"

	// AgoraEnvironmentTest is meant to be used in go tests
	AgoraEnvironmentTest AgoraEnvironment = "test"
)

var (
	// ErrBadEnvironmentVariableSet occurs when the AGORA_ENVIRONMENT environment variable is set to an invalid value
	ErrBadEnvironmentVariableSet = errors.New("environment variable AGORA_ENVIRONMENT was not 'prod', 'dev', or 'test'")
)

// FromEnvVariable will try to retrieve the environment variable AGORA_ENVIRONMENT. If the value is not 'prod',
// 'dev', or 'test', it will return an error
func FromEnvVariable() (AgoraEnvironment, error) {
	env := AgoraEnvironment(os.Getenv("AGORA_ENVIRONMENT"))
	if !env.IsValid() {
		return "", ErrBadEnvironmentVariableSet
	}
	return env, nil
}

// IsValid returns true if the AgoraEnvironment is valid.
func (env AgoraEnvironment) IsValid() bool {
	switch env {
	case AgoraEnvironmentProd, AgoraEnvironmentDev, AgoraEnvironmentTest:
		return true
	default:
		return false
	}
}
