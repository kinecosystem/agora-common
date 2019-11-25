package test

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/ory/dockertest"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
)

const (
	containerName    = "amazon/dynamodb-local"
	containerVersion = "1.11.477"
)

// StartDynamoDB starts a Docker container using the dynamodb-local image and returns a DynamoDB client for testing purposes.
func StartDynamoDB(pool *dockertest.Pool) (db dynamodbiface.ClientAPI, closeFunc func(), err error) {
	closeFunc = func() {}

	resource, err := pool.Run(containerName, containerVersion, nil)
	if err != nil {
		return nil, closeFunc, errors.Wrapf(err, "failed to start resource")
	}

	closeFunc = func() {
		if err := pool.Purge(resource); err != nil {
			logrus.StandardLogger().WithError(err).Warn("Failed to cleanup dynamodb resource")
		}
	}

	port := resource.GetPort("8000/tcp")

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, closeFunc, errors.Wrapf(err, "failed to load default aws config")
	}

	// Ensure that clients never reach out to a real system
	cfg.Region = "test-region-1"

	cfg.Credentials = aws.NewStaticCredentialsProvider("test", "test", "test")
	cfg.EndpointResolver = aws.ResolveWithEndpointURL(fmt.Sprintf("http://localhost:%s", port))

	return dynamodb.New(cfg), closeFunc, nil
}
