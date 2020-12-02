package sqs

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/ory/dockertest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqstest "github.com/kinecosystem/agora-common/aws/sqs/test"
	"github.com/kinecosystem/agora-common/taskqueue/model/task"
	"github.com/kinecosystem/agora-common/testutil"
)

var (
	sqsClient sqsiface.ClientAPI
)

func TestMain(m *testing.M) {
	testPool, err := dockertest.NewPool("")
	if err != nil {
		panic("Error creating docker pool:" + err.Error())
	}

	client, cleanUpSqs, err := sqstest.StartLocalSQS(testPool)
	if err != nil {
		panic("Error starting SQS image:" + err.Error())
	}
	defer cleanUpSqs()
	sqsClient = client

	defaultConfig.PollingInterval = time.Second
	defaultConfig.VisibilityTimeout = time.Second

	code := m.Run()
	cleanUpSqs()
	os.Exit(code)
}

func TestTaskQueue_Basic(t *testing.T) {
	queueName := fmt.Sprintf("%s%s", "test-queue-", uuid.New().String())
	setupQueue(t, queueName)
	defer deleteQueue(t, queueName)

	msgCh := make(chan task.Message, 100)
	defer close(msgCh)

	p, err := NewProcessor(queueName, sqsClient, func(ctx context.Context, msg *task.Message) error {
		select {
		case msgCh <- *msg:
		default:
			require.Fail(t, "task chan full")
		}
		return nil
	})
	require.NoError(t, err)
	defer p.Shutdown()

	expectedMsgs := map[string]struct{}{}
	for i := 0; i < 10; i++ {
		pb := &task.Message{RawValue: []byte("asdfasdf")}
		pbBytes, err := proto.Marshal(pb)
		require.NoError(t, err)

		msg := &task.Message{
			TypeName: proto.MessageName(pb),
			RawValue: pbBytes,
		}
		msgBytes, err := proto.Marshal(msg)
		require.NoError(t, err)

		expectedMsgs[string(msgBytes)] = struct{}{}
		require.NoError(t, p.Submit(context.Background(), msg))
	}

	// todo(metrics): 10 successes, no failures
	require.NoError(t, testutil.WaitFor(2*time.Second, 200*time.Millisecond, func() bool {
		// todo(metrics): 10 success
		return len(msgCh) == 10
	}))

	// Verify received msgs, note that ordering is not guaranteed
	for i := 0; i < 10; i++ {
		msg := <-msgCh

		msgBytes, err := proto.Marshal(&msg)
		require.NoError(t, err)

		_, ok := expectedMsgs[string(msgBytes)]
		require.True(t, ok)

		pb := &task.Message{}
		require.Equal(t, msg.TypeName, proto.MessageName(pb))
		require.NoError(t, proto.Unmarshal(msg.RawValue, pb))
		require.Equal(t, "asdfasdf", string(pb.RawValue))
	}

	require.Len(t, msgCh, 0)
}

func TestTaskQueue_RawB64Encoding(t *testing.T) {
	queueName := fmt.Sprintf("%s%s", "test-queue-", uuid.New().String())
	queueURL := setupQueue(t, queueName)
	defer deleteQueue(t, queueName)

	msgCh := make(chan task.Message, 5)
	defer close(msgCh)

	p, err := NewProcessor(queueName, sqsClient, func(ctx context.Context, msg *task.Message) error {
		select {
		case msgCh <- *msg:
		default:
			require.Fail(t, "task chan full")
		}
		return nil
	})
	require.NoError(t, err)
	defer p.Shutdown()

	wrapper := &task.Wrapper{
		Message: &task.Message{
			TypeName: "type",
			RawValue: []byte("message"),
		},
		SubmissionTime: ptypes.TimestampNow(),
	}

	rawWrapper, err := proto.Marshal(wrapper)
	require.NoError(t, err)

	_, err = sqsClient.SendMessageRequest(&sqs.SendMessageInput{
		MessageBody: aws.String(base64.RawURLEncoding.EncodeToString(rawWrapper)),
		QueueUrl:    aws.String(queueURL),
	}).Send(context.Background())
	require.NoError(t, err)

	// todo(metrics): confirm with metrics that a failure occured.

	msg := <-msgCh
	assert.True(t, proto.Equal(wrapper.Message, &msg))
}

func TestTaskQueue_InvalidTask(t *testing.T) {
	queueName := fmt.Sprintf("%s%s", "test-queue-", uuid.New().String())
	setupQueue(t, queueName)
	defer deleteQueue(t, queueName)

	msgCh := make(chan task.Message, 100)
	defer close(msgCh)
	p, err := NewProcessor(queueName, sqsClient, func(ctx context.Context, msg *task.Message) error {
		select {
		case msgCh <- *msg:
		default:
			require.Fail(t, "task chan full")
		}
		return nil
	})
	require.NoError(t, err)
	defer p.Shutdown()

	invalidMsgs := []*task.Message{
		{},
		{ // No type name
			RawValue: []byte("asdf"),
		},
		{ // Exceed limit
			TypeName: "asdf",
			RawValue: []byte(strings.Repeat("a", sqsByteLimit+1)),
		},
	}

	for _, m := range invalidMsgs {
		err := p.Submit(context.Background(), m)
		require.Error(t, err)
		t.Log(err)
	}

	// todo(metrics): verify metrics
}

func TestTaskQueue_TaskHandlerError(t *testing.T) {
	queueName := fmt.Sprintf("%s%s", "test-queue-", uuid.New().String())
	setupQueue(t, queueName)
	defer deleteQueue(t, queueName)

	msgCh := make(chan task.Message, 100)
	defer close(msgCh)

	first := true
	p, err := NewProcessor(queueName, sqsClient, func(ctx context.Context, msg *task.Message) error {
		// Fail first time
		if first {
			first = false
			return errors.New("error")
		}
		select {
		case msgCh <- *msg:
		default:
			require.Fail(t, "task chan full")
		}
		return nil
	})
	require.NoError(t, err)
	defer p.Shutdown()

	// Submit 1 message
	taskMsg := &task.Message{
		TypeName: "something",
		RawValue: []byte("asdf"),
	}
	require.NoError(t, p.Submit(context.Background(), taskMsg))

	// todo(metrics): verify submits

	// Wait for message to be successfully processed
	start := time.Now()
	require.NoError(t, testutil.WaitFor(2*time.Second, 200*time.Millisecond, func() bool {
		// todo(metrics): 1 success
		return len(msgCh) == 1
	}))
	end := time.Now()

	// Should have taken at least 1 second (visibility timeout) to successfully process
	// the task message on the second try
	require.True(t, end.Sub(start) >= 1*time.Second)

	receivedMsg := <-msgCh
	require.True(t, proto.Equal(taskMsg, &receivedMsg))

	// todo(metrics): verify 1 failure and 1 success
}

func TestTaskQueue_Submitter(t *testing.T) {
	queueName := fmt.Sprintf("%s%s", "test-queue-", uuid.New().String())
	setupQueue(t, queueName)
	defer deleteQueue(t, queueName)

	s, err := NewSubmitter(queueName, sqsClient)
	require.NoError(t, err)

	expectedMsgs := make(map[string]struct{})
	for i := 0; i < 10; i++ {
		msg := &task.Message{
			TypeName: "something",
			RawValue: []byte(fmt.Sprintf("hello%d", i)),
		}
		msgBytes, err := proto.Marshal(msg)
		require.NoError(t, err)
		expectedMsgs[string(msgBytes)] = struct{}{}
		require.NoError(t, s.Submit(context.Background(), msg))
	}

	// No task messages should be consumed
	time.Sleep(500 * time.Millisecond)
	// todo(metrics): verify no successes or failures

	msgCh := make(chan task.Message, 100)
	defer close(msgCh)
	p, err := NewProcessor(queueName, sqsClient, func(ctx context.Context, msg *task.Message) error {
		select {
		case msgCh <- *msg:
		default:
			require.Fail(t, "task chan full")
		}
		return nil
	})
	require.NoError(t, err)
	defer p.Shutdown()

	// Processor should consume tasks
	// todo(metrics): 10 successes, no failures
	require.NoError(t, testutil.WaitFor(2*time.Second, 200*time.Millisecond, func() bool {
		// todo(metrics): 2 success
		return len(msgCh) == 10
	}))

	for i := 0; i < 10; i++ {
		msg := <-msgCh

		msgBytes, err := proto.Marshal(&msg)
		require.NoError(t, err)

		_, ok := expectedMsgs[string(msgBytes)]
		require.True(t, ok)
	}

	require.Len(t, msgCh, 0)
}

func TestTaskQueue_SubmitterBatch(t *testing.T) {
	queueName := fmt.Sprintf("%s%s", "test-queue-", uuid.New().String())
	setupQueue(t, queueName)
	defer deleteQueue(t, queueName)

	s, err := NewSubmitter(queueName, sqsClient)
	require.NoError(t, err)

	expectedMsgs := make(map[string]struct{})
	msgs := make([]*task.Message, 25)

	for i := 0; i < 25; i++ {
		msg := &task.Message{
			TypeName: "something",
			RawValue: []byte(fmt.Sprintf("hello%d", i)),
		}
		msgs[i] = msg
		msgBytes, err := proto.Marshal(msg)
		require.NoError(t, err)
		expectedMsgs[string(msgBytes)] = struct{}{}
	}

	require.NoError(t, s.SubmitBatch(context.Background(), msgs))

	// No task messages should be consumed
	time.Sleep(500 * time.Millisecond)
	// todo(metrics): verify no successes or failures

	msgCh := make(chan task.Message, 100)
	defer close(msgCh)
	p, err := NewProcessor(queueName, sqsClient, func(ctx context.Context, msg *task.Message) error {
		select {
		case msgCh <- *msg:
		default:
			require.Fail(t, "task chan full")
		}
		return nil
	})
	require.NoError(t, err)
	defer p.Shutdown()

	// Processor should consume tasks
	// todo(metrics): 10 successes, no failures
	require.NoError(t, testutil.WaitFor(2*time.Second, 200*time.Millisecond, func() bool {
		// todo(metrics): 2 success
		return len(msgCh) == 25
	}))

	for i := 0; i < 25; i++ {
		msg := <-msgCh

		msgBytes, err := proto.Marshal(&msg)
		require.NoError(t, err)

		_, ok := expectedMsgs[string(msgBytes)]
		require.True(t, ok)
	}

	require.Len(t, msgCh, 0)
}

func TestTaskQueue_VisibilityTimeoutExceeded(t *testing.T) {
	queueName := fmt.Sprintf("%s%s", "test-queue-", uuid.New().String())
	setupQueue(t, queueName)
	defer deleteQueue(t, queueName)

	chanMu := sync.RWMutex{} // To appease the race detector
	msgsChan := make(chan task.Message, 100)
	defer close(msgsChan)

	var first int32 = 1
	p, err := NewProcessor(queueName, sqsClient, func(ctx context.Context, msg *task.Message) error {
		if atomic.CompareAndSwapInt32(&first, 1, 0) {
			// First attempt will timeout
			time.Sleep(1400 * time.Millisecond)
		}

		chanMu.Lock()
		select {
		case msgsChan <- *msg:
		default:
			require.Fail(t, "task chan full")
		}
		chanMu.Unlock()

		return nil
	})
	require.NoError(t, err)
	defer p.Shutdown()

	// Submit one task
	taskMsg := &task.Message{
		TypeName: "something",
		RawValue: []byte("asdf"),
	}
	require.NoError(t, p.Submit(context.Background(), taskMsg))

	// todo(metrics): assert 1 submission

	// Expect message to be processed 2 times since the first task exceeded visibility timeout
	require.NoError(t, testutil.WaitFor(4*time.Second, 500*time.Millisecond, func() bool {
		chanMu.RLock()
		received := len(msgsChan)
		chanMu.RUnlock()

		// todo(metrics): assert 1 success, 1 failure
		return received == 2
	}))

	// Only one attempt marked as success
	// todo(metrics): assert 1 success, 1 failure
}

func TestTaskQueue_VisibilityTimeoutExtension(t *testing.T) {
	queueName := fmt.Sprintf("%s%s", "test-queue-", uuid.New().String())
	setupQueue(t, queueName)
	defer deleteQueue(t, queueName)

	msgsChan := make(chan task.Message, 100)
	defer close(msgsChan)

	p, err := NewProcessor(queueName, sqsClient, func(ctx context.Context, msg *task.Message) error {
		// Task exceeds one visibility timeout block
		time.Sleep(1400 * time.Millisecond)
		select {
		case msgsChan <- *msg:
		default:
			require.Fail(t, "task chan full")
		}
		return nil
	}, WithVisibilityExtensionEnabled(true))
	require.NoError(t, err)
	defer p.Shutdown()

	// Submit one task
	taskMsg := &task.Message{
		TypeName: "something",
		RawValue: []byte("asdf"),
	}
	require.NoError(t, p.Submit(context.Background(), taskMsg))
	// todo(metrics): 1 submission

	// Expect message to be processed 1 time
	require.NoError(t, testutil.WaitFor(3*time.Second, 500*time.Millisecond, func() bool {
		// todo(metrics): 1 success
		return len(msgsChan) == 1
	}))

	// Only one attempt marked as success
	// todo(metrics): 1 success, 0 failures
}

func setupQueue(t *testing.T, queueName string) string {
	resp, err := sqsClient.GetQueueUrlRequest(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	}).Send(context.Background())
	if err != nil {
		resp, err := sqsClient.CreateQueueRequest(&sqs.CreateQueueInput{
			QueueName: aws.String(queueName),
		}).Send(context.Background())
		require.NoError(t, err)
		return aws.StringValue(resp.QueueUrl)
	}
	return aws.StringValue(resp.QueueUrl)
}

func deleteQueue(t *testing.T, queueName string) {
	resp, err := sqsClient.GetQueueUrlRequest(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	}).Send(context.Background())
	require.NoError(t, err)

	// Clear out the queue
	_, err = sqsClient.DeleteQueueRequest(&sqs.DeleteQueueInput{
		QueueUrl: resp.QueueUrl,
	}).Send(context.Background())
	require.NoError(t, err)
}
