package kin

import (
	"errors"
	"net/http"

	"github.com/stellar/go/clients/horizonclient"

	agoraenv "github.com/kinecosystem/agora-common/env"
	"github.com/kinecosystem/agora-common/kin/network"
	"github.com/kinecosystem/go/build"
	"github.com/kinecosystem/go/clients/horizon"
)

const (
	KinAssetCode = "KIN"

	// prodHorizonURL is the URL of the production Kin Horizon server
	prodHorizonURL = "https://horizon.kinfederation.com"
	// testHorizonURL is the URL of the test Kin Horizon server
	testHorizonURL = "https://horizon-testnet.kininfrastructure.com"

	// kin2ProdHorizonURL is the URL of the production Kin 2 Horizon server
	kin2ProdHorizonURL = "https://horizon-kin-ecosystem.kininfrastructure.com"
	// kin2TestHorizonURL is the URL of the test Kin 2 Horizon server
	kin2TestHorizonURL = "https://horizon-playground.kininfrastructure.com"

	// prodHorizonPassphrase is the passphrase for the production Kin network
	prodHorizonPassphrase = "Kin Mainnet ; December 2018"
	// testHorizonPassphrase is the passphrase for the test Kin network
	testHorizonPassphrase = "Kin Testnet ; December 2018"

	// kin2ProdPassphrase is the passphrase for the production Kin 2 network
	kin2ProdPassphrase = "Public Global Kin Ecosystem Network ; June 2018"
	// kin2TestPassphrase is the passphrase for the test Kin 2 network
	kin2TestPassphrase = "Kin Playground Network ; June 2018"

	// kin2ProdIssuer is the Kin 2 issuer address on the production Kin 2 network
	kin2ProdIssuer = "GDF42M3IPERQCBLWFEZKQRK77JQ65SCKTU3CW36HZVCX7XX5A5QXZIVK"
	// kin2TestIssuer is the Kin 2 issuer address on the test Kin 2 network
	kin2TestIssuer = "GBC3SG6NGTSZ2OMH3FFGB7UVRQWILW367U4GSOOF4TFSZONV42UJXUH7"
)

var (
	// kinProdHorizonClient is the Horizon Client that should be used to interact with the production Kin network
	kinProdHorizonClient = &horizon.Client{
		URL:  prodHorizonURL,
		HTTP: http.DefaultClient,
	}

	// kinTestHorizonClient is the Horizon Client that should be used to interact with the test Kin network
	kinTestHorizonClient = &horizon.Client{
		URL:  testHorizonURL,
		HTTP: http.DefaultClient,
	}

	// kin2ProdHorizonClient is the Horizon Client that should be used to interact with the production Kin 2 network
	kin2ProdHorizonClient = &horizon.Client{
		URL:  kin2ProdHorizonURL,
		HTTP: http.DefaultClient,
	}

	// kin2TestHorizonClient is the Horizon Client that should be used to interact with the test Kin 2 network
	kin2TestHorizonClient = &horizon.Client{
		URL:  kin2TestHorizonURL,
		HTTP: http.DefaultClient,
	}

	// kinProdHorizonClientV2 is the Horizon Client (from stellar) that should be used to interact with the
	// production Kin network.
	kinProdHorizonClientV2 = &horizonclient.Client{
		HorizonURL: prodHorizonURL,
		HTTP:       http.DefaultClient,
	}

	// kinTestHorizonClientV2 is the Horizon Client (from stellar) that should be used to interact with the
	// test Kin network.
	kinTestHorizonClientV2 = &horizonclient.Client{
		HorizonURL: testHorizonURL,
		HTTP:       http.DefaultClient,
	}

	// kin2ProdHorizonClientV2 is the Horizon Client (from stellar) that should be used to interact with the
	// production Kin 2 network.
	kin2ProdHorizonClientV2 = &horizonclient.Client{
		HorizonURL: kin2ProdHorizonURL,
		HTTP:       http.DefaultClient,
	}

	// kin2TestHorizonClientV2 is the Horizon Client (from stellar) that should be used to interact with the
	// test Kin 2 network.
	kin2TestHorizonClientV2 = &horizonclient.Client{
		HorizonURL: kin2TestHorizonURL,
		HTTP:       http.DefaultClient,
	}

	// prodNetwork is the Network modifier that should be used in transactions on the production Kin network
	prodNetwork = build.Network{Passphrase: prodHorizonPassphrase}
	// testNetwork is the Network modifier that should be used in transactions on the test Kin network
	testNetwork     = build.Network{Passphrase: testHorizonPassphrase}
	// kin2ProdNetwork is the Network modifier that should be used in transactions on the production Kin 2 network
	kin2ProdNetwork = build.Network{Passphrase: kin2ProdPassphrase}
	// kin2TestNetwork is the Network modifier that should be used in transactions on the test Kin 2 network
	kin2TestNetwork = build.Network{Passphrase: kin2TestPassphrase}
)

var (
	// ErrInvalidKinNetwork occurs when an invalid KinNetwork is provided
	ErrInvalidKinNetwork = errors.New("KinNetwork was not 'mainnet' or 'testnet'")
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

// GetClientV2 returns the default stellar based Horizon client based on which environment the application is running in.
//
// The stellar based client offers some additional niceties, notably around retrieving
// transaction history. It's generally considered to be a better client, however, the
// functionality _may_ have some divergent behaviour from the kin fork. Therefore, any
// use of this client should be tested thoroughly.
func GetClientV2() (client *horizonclient.Client, err error) {
	env, err := agoraenv.FromEnvVariable()
	if err != nil {
		return nil, err
	}

	switch env {
	case agoraenv.AgoraEnvironmentProd:
		return kinProdHorizonClientV2, nil
	default:
		return kinTestHorizonClientV2, nil
	}
}

// GetKin2Client returns the default Kin 2 Horizon client based on which environment the environment is running in
func GetKin2Client() (client *horizon.Client, err error) {
	env, err := agoraenv.FromEnvVariable()
	if err != nil {
		return nil, err
	}

	switch env {
	case agoraenv.AgoraEnvironmentProd:
		return kin2ProdHorizonClient, nil
	default:
		return kin2TestHorizonClient, nil
	}
}

// GetKin2ClientV2 returns the default stellar-based Kin 2 Horizon client based on which environment the application
// is running in.
//
// The stellar based client offers some additional niceties, notably around retrieving
// transaction history. It's generally considered to be a better client, however, the
// functionality _may_ have some divergent behaviour from the kin fork. Therefore, any
// use of this client should be tested thoroughly.
func GetKin2ClientV2() (client *horizonclient.Client, err error) {
	env, err := agoraenv.FromEnvVariable()
	if err != nil {
		return nil, err
	}

	switch env {
	case agoraenv.AgoraEnvironmentProd:
		return kin2ProdHorizonClientV2, nil
	default:
		return kin2TestHorizonClientV2, nil
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

// GetKin2Network returns the default Kin 2 Network modifier based on which environment the application is running in.
func GetKin2Network() (network build.Network, err error) {
	env, err := agoraenv.FromEnvVariable()
	if err != nil {
		return build.Network{}, err
	}

	switch env {
	case agoraenv.AgoraEnvironmentProd:
		return kin2ProdNetwork, nil
	default:
		return kin2TestNetwork, nil
	}
}

// GetClientByKinNetwork returns a Horizon client for the provided Kin network
func GetClientByKinNetwork(net network.KinNetwork) (client *horizon.Client, err error) {
	if !net.IsValid() {
		return nil, ErrInvalidKinNetwork
	}

	switch net {
	case network.MainNetwork:
		return kinProdHorizonClient, nil
	default:
		return kinTestHorizonClient, nil
	}
}

// GetNetworkByKinNetwork returns a Network modifier for the provided Kin network
func GetNetworkByKinNetwork(net network.KinNetwork) (buildNetwork build.Network, err error) {
	if !net.IsValid() {
		return build.Network{}, ErrInvalidKinNetwork
	}

	switch net {
	case network.MainNetwork:
		return prodNetwork, nil
	default:
		return testNetwork, nil
	}
}

// GetKin2Issuer returns the Kin issuer address based on which environment the application is running in.
func GetKin2Issuer() (issuer string, err error) {
	env, err := agoraenv.FromEnvVariable()
	if err != nil {
		return "", err
	}

	switch env {
	case agoraenv.AgoraEnvironmentProd:
		return kin2ProdIssuer, nil
	default:
		return kin2TestIssuer, nil
	}
}
