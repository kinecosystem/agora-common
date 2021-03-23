package createaccount

// Request contains the body of a create account request.
type Request struct {
	KinVersion int `json:"kin_version"`
	// SolanaTransaction is a base64-encoded Solana transaction
	SolanaTransaction []byte `json:"solana_transaction"`
}

// SuccessResponse represents a 200 OK response to a create account request.
type SuccessResponse struct {
	// Signature is a base64-encoded transaction signature.
	Signature []byte `json:"signature"`
}
