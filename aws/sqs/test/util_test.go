package test

import (
	"context"
	"testing"

	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func TestLocalSQS(t *testing.T) {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	sqsClient, cleanup, err := StartLocalSQS(pool)
	require.NoError(t, err)
	defer cleanup()

	listResp, err := sqsClient.ListQueuesRequest(&sqs.ListQueuesInput{}).Send(context.Background())
	require.NoError(t, err)
	assert.Empty(t, listResp.QueueUrls)

	// Create a queue
	createResp, err := sqsClient.CreateQueueRequest(&sqs.CreateQueueInput{
		QueueName: aws.String("test-queue"),
	}).Send(context.Background())
	require.NoError(t, err)
	queueURL := aws.StringValue(createResp.QueueUrl)

	listResp, err = sqsClient.ListQueuesRequest(&sqs.ListQueuesInput{}).Send(context.Background())
	require.NoError(t, err)
	assert.Len(t, listResp.QueueUrls, 1)
	assert.Equal(t, queueURL, listResp.QueueUrls[0])

	// Send a msg
	msg := "Message in a bottle"
	_, err = sqsClient.SendMessageRequest(&sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(msg),
	}).Send(context.Background())
	require.NoError(t, err)

	// Recv a msg
	recv, err := sqsClient.ReceiveMessageRequest(&sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: aws.Int64(1),
		VisibilityTimeout:   aws.Int64(5),
		WaitTimeSeconds:     aws.Int64(5),
	}).Send(context.Background())
	require.NoError(t, err)
	assert.Len(t, recv.Messages, 1)
	assert.Equal(t, msg, aws.StringValue(recv.Messages[0].Body))
}
