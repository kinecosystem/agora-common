package network

type KinNetwork string

const (
	// MainNetwork is the production Kin network
	MainNetwork KinNetwork = "mainnet"

	// TestNetwork is the test Kin network
	TestNetwork KinNetwork = "testnet"
)

// IsValid returns true if the KinNetwork is valid.
func (network KinNetwork) IsValid() bool {
	switch network {
	case MainNetwork, TestNetwork:
		return true
	default:
		return false
	}
}
