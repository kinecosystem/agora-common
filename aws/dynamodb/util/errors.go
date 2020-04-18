package util

import (
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// IsConditionalCheckFailed returns whether or not the error indicates
// the that the conditional expression failed.
func IsConditionalCheckFailed(err error) bool {
	if aErr, ok := err.(awserr.Error); ok {
		return aErr.Code() == dynamodb.ErrCodeConditionalCheckFailedException
	}

	return false
}

// MapConditionalCheckFailed returns the desired error if the provied
// error is a conditional expression failed error. Otherwise the
// original error is returned.
func MapConditionalCheckFailed(err, desired error) error {
	if IsConditionalCheckFailed(err) {
		return desired
	}

	return err
}
