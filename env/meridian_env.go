package env

import (
	"errors"
	"os"
)

// MeridianEnvironment is used to determine which environment the application is currently running in
type MeridianEnvironment string

var (
	// MeridianEnvironmentProd is the production environment
	MeridianEnvironmentProd MeridianEnvironment = "prod"

	// MeridianEnvironmentDev is the development environment
	MeridianEnvironmentDev MeridianEnvironment = "dev"

	// MeridianEnvironmentTest is meant to be used in go tests
	MeridianEnvironmentTest MeridianEnvironment = "test"
)

var (
	// ErrBadEnvironmentVariableSet occurs when the MERIDIAN_ENVIRONMENT environment variable is set to an invalid value
	ErrBadEnvironmentVariableSet = errors.New("environment variable MERIDIAN_ENVIRONMENT was not 'prod', 'dev', or 'test'")
)

// FromEnvVariable will try to retrieve the environment variable MERIDIAN_ENVIRONMENT. If the value is not 'prod',
// 'dev', or 'test', it will return an error
func FromEnvVariable() (MeridianEnvironment, error) {
	env := MeridianEnvironment(os.Getenv("MERIDIAN_ENVIRONMENT"))
	if !env.IsValid() {
		return "", ErrBadEnvironmentVariableSet
	}
	return env, nil
}

// IsValid returns true if the MeridianEnvironment is valid.
func (env MeridianEnvironment) IsValid() bool {
	switch env {
	case MeridianEnvironmentProd, MeridianEnvironmentDev, MeridianEnvironmentTest:
		return true
	default:
		return false
	}
}
