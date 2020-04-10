package test

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
	"github.com/ory/dockertest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/kinecosystem/agora-common/retry"
	"github.com/kinecosystem/agora-common/retry/backoff"
)

const (
	containerName    = "vsouza/sqs-local"
	containerVersion = "latest"
)

// StartLocalSQS starts a local dockerized SQS server for testing.
func StartLocalSQS(pool *dockertest.Pool) (sqsiface.ClientAPI, func(), error) {
	closeFunc := func() {}

	resource, err := pool.Run(containerName, containerVersion, nil)
	if err != nil {
		return nil, closeFunc, errors.Wrapf(err, "failed to start sqs resource")
	}

	closeFunc = func() {
		if err := pool.Purge(resource); err != nil {
			logrus.StandardLogger().WithError(err).Warn("Failed to cleanup sqs resource")
		}
	}

	port := resource.GetPort("9324/tcp")

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, closeFunc, errors.Wrapf(err, "failed to load default aws config")
	}

	// Ensure that clients never reach out to a real system
	cfg.Region = "test-region-1"
	cfg.Credentials = aws.NewStaticCredentialsProvider("test", "test", "test")
	cfg.EndpointResolver = aws.ResolveWithEndpointURL(fmt.Sprintf("http://localhost:%s", port))

	client := sqs.New(cfg)

	_, err = retry.Retry(
		func() error {
			_, err := client.ListQueuesRequest(&sqs.ListQueuesInput{}).Send(context.Background())
			return err
		},
		retry.Limit(20),
		retry.Backoff(backoff.Constant(500*time.Millisecond), 500*time.Millisecond),
	)
	if err != nil {
		return nil, closeFunc, errors.Wrapf(err, "timeout waiting for local SQS to become responsive")
	}

	return client, closeFunc, nil
}
