package sqs

import (
	"context"
	"encoding/base64"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/kinecosystem/agora-common/taskqueue"
	"github.com/kinecosystem/agora-common/taskqueue/model/task"
)

const (
	sqsByteLimit = 262144
)

type queue struct {
	log      *logrus.Entry
	conf     config
	sqs      sqsiface.ClientAPI
	queueURL string
	handler  taskqueue.Handler

	wg sync.WaitGroup

	shutdownCh   chan struct{}
	shutdownOnce sync.Once
}

func NewProcessorCtor(queueName string, sqsClient sqsiface.ClientAPI, opts ...Option) taskqueue.ProcessorCtor {
	return func(handler taskqueue.Handler) (taskqueue.Processor, error) {
		return NewProcessor(queueName, sqsClient, handler, opts...)
	}
}

func NewProcessor(queueName string, sqsClient sqsiface.ClientAPI, handler taskqueue.Handler, opts ...Option) (taskqueue.Processor, error) {
	if handler == nil {
		return nil, errors.Errorf("handler is nil")
	}

	return newQueue(queueName, sqsClient, handler, opts...)
}

func NewSubmitter(queueName string, sqsClient sqsiface.ClientAPI, opts ...Option) (taskqueue.Submitter, error) {
	return newQueue(queueName, sqsClient, nil, opts...)
}

func newQueue(queueName string, sqsClient sqsiface.ClientAPI, handler taskqueue.Handler, opts ...Option) (*queue, error) {
	q := &queue{
		log: logrus.StandardLogger().WithFields(logrus.Fields{
			"type":  "taskqueue/sqs",
			"queue": queueName,
		}),
		conf:       defaultConfig,
		sqs:        sqsClient,
		shutdownCh: make(chan struct{}),
		handler:    handler,
	}

	for _, o := range opts {
		o(&q.conf)
	}

	resp, err := sqsClient.GetQueueUrlRequest(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	}).Send(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get queue url")
	}
	q.queueURL = aws.StringValue(resp.QueueUrl)

	if handler != nil {
		q.wg.Add(q.conf.TaskConcurrency)
		for i := 0; i < q.conf.TaskConcurrency; i++ {
			go func(id int) {
				q.taskWorker(id)
			}(i)
		}
	}

	return q, nil
}

// Submit implements taskqueue.Submitter.Submit,
func (q *queue) Submit(ctx context.Context, msg *task.Message) error {
	select {
	case <-q.shutdownCh:
		return errors.New("queue shutting down")
	default:
	}

	msgBody, err := marshalTask(msg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal task")
	}

	_, err = q.sqs.SendMessageRequest(&sqs.SendMessageInput{
		QueueUrl:    aws.String(q.queueURL),
		MessageBody: aws.String(msgBody),
	}).Send(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to submit task")
	}

	return nil
}

func (q *queue) Shutdown() {
	q.shutdownOnce.Do(func() {
		log := q.log.WithField("method", "Shutdown")
		close(q.shutdownCh)

		gracePeriod := q.conf.VisibilityTimeout
		if ok := waitForGroup(&q.wg, gracePeriod); !ok {
			log.Warnf("workers did not fully shutdown within the grace period %s", gracePeriod)
		}
	})
}

func (q *queue) taskWorker(id int) {
	log := q.log.WithField("worker_id", id)
	log.Debug("worker starting")
	defer func() {
		q.wg.Done()
		log.Info("worker stopped")
	}()

	for {
		select {
		case <-q.shutdownCh:
			return
		default:
		}

		// todo(config?): temp disable at runtime.

		resp, err := q.sqs.ReceiveMessageRequest(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(q.queueURL),
			MaxNumberOfMessages: aws.Int64(1),
			VisibilityTimeout:   aws.Int64(int64(q.conf.VisibilityTimeout.Seconds())),
			WaitTimeSeconds:     aws.Int64(int64(q.conf.PollingInterval.Seconds())),
		}).Send(context.Background())
		if err != nil {
			log.WithError(err).Warn("failed to poll for tasks")
			time.Sleep(5 * time.Second)
			continue
		}

		for _, msg := range resp.Messages {
			receiptHandle := aws.StringValue(msg.ReceiptHandle)

			if msg.Body == nil {
				log.WithField("message", msg.String()).Info("got empty message, deleting from queue")
				if err := q.deleteMessage(receiptHandle); err != nil {
					log.WithError(err).Warn("failed to delete empty message from queue")
				}
				continue
			}

			wrapper, err := unmarshalTask(aws.StringValue(msg.Body))
			if err != nil {
				log.WithError(err).Warn("failed to unmarshal message")
				if err := q.deleteMessage(receiptHandle); err != nil {
					log.WithError(err).Warn("failed to delete invalid message from queue")
				}
				continue
			}

			// todo(metrics): add metric for now() - submissionTime

			log.WithField("task", wrapper.String()).Trace("received task message")
			if err := q.processTask(receiptHandle, q.conf.VisibilityTimeout, wrapper.Message); err != nil {
				// handler is expected to do logging
				// todo(metrics): meter failed processing
			} else {
				if err := q.deleteMessage(receiptHandle); err != nil {
					log.WithError(err).Warn("failed to delete completed message from queue")
				}

				// todo(metrics): add metrics for success + timing
			}
		}
	}
}

func (q *queue) processTask(handle string, visibilityTimeout time.Duration, msg *task.Message) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// todo(metrics): add timing?
	result := make(chan error)
	go func() {
		result <- q.handler(ctx, msg)
	}()

	// extend the visibility timeout when 80% of the timeout has elapsed to be safe.
	keepAliveInterval := visibilityTimeout / 5 * 4

	for ext := 0; ext < q.conf.MaxVisibilityExtensions; ext++ {
		select {
		case <-q.shutdownCh:
			return errors.New("processor shutting down, not waiting for task")
		case err := <-result:
			return err

		case <-time.After(keepAliveInterval):
			if !q.conf.VisibilityExtensionEnabled {
				return errors.Errorf("task handler timed out after %v (80 percent of visibility timeout)", keepAliveInterval)
			}

			if err := q.extendVisibilityTimeout(handle, visibilityTimeout); err != nil {
				// just give up, let the task become visible and be processed later
				return errors.Wrap(err, "failed to extend visibility timeout for task")
			}
		}
	}

	return errors.Errorf("max visibility extensions (%d) exceeded, not waiting for task", q.conf.MaxVisibilityExtensions)
}

func (q *queue) extendVisibilityTimeout(handle string, timeout time.Duration) error {
	_, err := q.sqs.ChangeMessageVisibilityRequest(&sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(q.queueURL),
		ReceiptHandle:     aws.String(handle),
		VisibilityTimeout: aws.Int64(int64(timeout.Seconds())),
	}).Send(context.Background())
	return err
}

func (q *queue) deleteMessage(handle string) error {
	_, err := q.sqs.DeleteMessageRequest(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.queueURL),
		ReceiptHandle: aws.String(handle),
	}).Send(context.Background())
	return err
}

func waitForGroup(wg *sync.WaitGroup, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		return true
	case <-ctx.Done():
		return false
	}
}

func marshalTask(msg *task.Message) (string, error) {
	if msg == nil {
		return "", errors.Errorf("task message is nil")
	}

	if err := msg.Validate(); err != nil {
		return "", err
	}

	taskWrapper := &task.Wrapper{
		Message:        msg,
		SubmissionTime: ptypes.TimestampNow(),
	}

	bytes, err := proto.Marshal(taskWrapper)
	if err != nil {
		return "", err
	}

	taskBody := base64.URLEncoding.EncodeToString(bytes)
	if len(taskBody) > sqsByteLimit {
		return "", errors.Errorf("encoded task payload size exceeded SQS limit (%d/%d)", len(taskBody), sqsByteLimit)
	}

	return taskBody, nil
}

func unmarshalTask(body string) (*task.Wrapper, error) {
	bytes, err := base64.URLEncoding.DecodeString(body)
	if err != nil {
		// Attempt to decode without padding, which adds compat with Java since
		// the Apache implementation we use defaults to using no padding.
		bytes, err = base64.RawURLEncoding.DecodeString(body)
		if err != nil {
			return nil, err
		}
	}

	wrapper := &task.Wrapper{}
	if err := proto.Unmarshal(bytes, wrapper); err != nil {
		return nil, err
	}

	if err := wrapper.Validate(); err != nil {
		return nil, err
	}

	return wrapper, nil
}
