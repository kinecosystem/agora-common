package kin

import (
	"net/http"

	agoraenv "github.com/kinecosystem/agora-common/env"
	"github.com/kinecosystem/go/build"
	"github.com/kinecosystem/go/clients/horizon"
)

const (
	// prodHorizonUrl is the URL of the production Kin Horizon server
	prodHorizonUrl = "https://horizon.kinfederation.com"
	// testHorizonUrl is the URL of the test Kin Horizon server
	testHorizonUrl = "https://horizon-testnet.kininfrastructure.com"

	// prodHorizonPassphrase is the passphrase for the production Kin network
	prodHorizonPassphrase = "Kin Mainnet ; December 2018"
	// testHorizonPassphrase is the passphrase for the test Kin network
	testHorizonPassphrase = "Kin Testnet ; December 2018"
)

var (
	// kinProdHorizonClient is the Horizon Client that should be used to interact with the production Kin network
	kinProdHorizonClient = &horizon.Client{
		URL:  prodHorizonUrl,
		HTTP: http.DefaultClient,
	}

	// kinTestHorizonClient is the Horizon Client that should be used to interact with the test Kin network
	kinTestHorizonClient = &horizon.Client{
		URL:  testHorizonUrl,
		HTTP: http.DefaultClient,
	}

	// prodNetwork is the Network modifier that should be used in transactions on the production Kin network
	prodNetwork = build.Network{Passphrase: prodHorizonPassphrase}
	// testNetwork is the Network modifier that should be used in transactions on the test Kin network
	testNetwork = build.Network{Passphrase: testHorizonPassphrase}
)

// GetClient returns the default Horizon client based on which environment the application is running in.
func GetClient() (client *horizon.Client, err error) {
	env, err := agoraenv.FromEnvVariable()
	if err != nil {
		return nil, err
	}

	switch env {
	case agoraenv.AgoraEnvironmentProd:
		return kinProdHorizonClient, nil
	default:
		return kinTestHorizonClient, nil
	}
}

// GetNetwork returns the default Network modifier based on which environment the application is running in.
func GetNetwork() (network build.Network, err error) {
	env, err := agoraenv.FromEnvVariable()
	if err != nil {
		return build.Network{}, err
	}

	switch env {
	case agoraenv.AgoraEnvironmentProd:
		return prodNetwork, nil
	default:
		return testNetwork, nil
	}
}
