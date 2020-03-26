package test

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/ory/dockertest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"mfycheng.dev/retry"
	"mfycheng.dev/retry/backoff"
)

const (
	repository    = "adobe/s3mock"
	tag = "latest"
)

// StartS3 starts a mock S3 dockerized server
func StartS3(pool *dockertest.Pool) (s3iface.ClientAPI, func(), error) {
	closeFunc := func() {}

	resource, err := pool.Run(repository, tag, nil)

	if err != nil {
		return nil, closeFunc, errors.Wrapf(err, "failed to start resource")
	}

	closeFunc = func() {
		if err := pool.Purge(resource); err != nil {
			logrus.StandardLogger().WithError(err).Warn("Failed to cleanup dynamodb resource")
		}
	}

	address := resource.GetHostPort("9090/tcp")

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, closeFunc, errors.Wrapf(err, "failed to load default aws config")
	}

	// Ensure that clients never reach out to a real system
	cfg.Region = "test-region-1"
	cfg.Credentials = aws.NewStaticCredentialsProvider("test", "test", "test")
	cfg.EndpointResolver = aws.ResolveWithEndpointURL(fmt.Sprintf("http://%s", address))

	client := s3.New(cfg)
	client.ForcePathStyle = true

	_, err = retry.Retry(
		func() error {
			_, err := client.ListBucketsRequest(&s3.ListBucketsInput{}).Send(context.Background())
			return err
		},
		retry.Limit(20),
		retry.Backoff(backoff.Constant(500*time.Millisecond), 500*time.Second),
	)
	if err != nil {
		return nil, closeFunc, errors.Wrap(err, "timed out waiting for s3 container to become available")
	}

	return client, closeFunc, nil
}
