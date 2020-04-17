package kin

import (
	commonpb "github.com/kinecosystem/kin-api/genproto/common/v3"
)

// GetAgoraDataTransactionType returns the appropriate AgoraData_TransactionType based on the transaction type in
// the provided kin memo.
func GetAgoraDataTransactionType(memo Memo) commonpb.AgoraData_TransactionType {
	switch memo.TransactionType() {
	case 1:
		return commonpb.AgoraData_EARN
	case 2:
		return commonpb.AgoraData_SPEND
	case 3:
		return commonpb.AgoraData_P2P
	default:
		return commonpb.AgoraData_UNKNOWN
	}
}
