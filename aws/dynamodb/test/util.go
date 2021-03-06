package test

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	dynamodbv1 "github.com/aws/aws-sdk-go/service/dynamodb"
	dynamodbifacev1 "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/ory/dockertest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/kinecosystem/agora-common/retry"
	"github.com/kinecosystem/agora-common/retry/backoff"
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

	address := resource.GetHostPort("8000/tcp")

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, closeFunc, errors.Wrapf(err, "failed to load default aws config")
	}

	// Ensure that clients never reach out to a real system
	cfg.Region = "test-region-1"

	cfg.Credentials = aws.NewStaticCredentialsProvider("test", "test", "test")
	cfg.EndpointResolver = aws.ResolveWithEndpointURL(fmt.Sprintf("http://%s", address))

	db = dynamodb.New(cfg)

	_, err = retry.Retry(
		func() error {
			_, err := db.ListTablesRequest(&dynamodb.ListTablesInput{}).Send(context.Background())
			return err
		},
		retry.Limit(20),
		retry.Backoff(backoff.Constant(500*time.Millisecond), 500*time.Second),
	)
	if err != nil {
		return nil, closeFunc, errors.Wrap(err, "timed out waiting for dynamodb container to become available")
	}

	return db, closeFunc, nil
}

// StartDynamoDBV1 starts a Docker container using the dynamodb-local image and returns a V1 DynamoDB client for testing purposes.
func StartDynamoDBV1(pool *dockertest.Pool) (db dynamodbifacev1.DynamoDBAPI, closeFunc func(), err error) {
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

	address := resource.GetHostPort("8000/tcp")

	sess, err := session.NewSession()
	if err != nil {
		return nil, closeFunc, errors.Wrapf(err, "failed to load default aws config")
	}

	resolver := func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		return endpoints.ResolvedEndpoint{
			URL: fmt.Sprintf("http://%s", address),
		}, nil
	}

	// Ensure that clients never reach out to a real system
	sess.Config.Region = aws.String("test-region-1")
	sess.Config.Credentials = credentials.NewStaticCredentials("test", "test", "test")
	sess.Config.EndpointResolver = endpoints.ResolverFunc(resolver)

	db = dynamodbv1.New(sess)

	_, err = retry.Retry(
		func() error {
			req, _ := db.ListTablesRequest(&dynamodbv1.ListTablesInput{})
			return req.Send()
		},
		retry.Limit(20),
		retry.Backoff(backoff.Constant(500*time.Millisecond), 500*time.Second),
	)
	if err != nil {
		return nil, closeFunc, errors.Wrap(err, "timed out waiting for dynamodb container to become available")
	}

	return db, closeFunc, nil
}
