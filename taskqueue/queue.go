package taskqueue

import (
	"context"

	"github.com/kinecosystem/agora-common/taskqueue/model/task"
)

// Handler is a handler for a message received from a task queue.
//
// If the task is long-lived, the task should periodically check the given context and stop
// if it is cancelled.
//
// If the task did not complete (and should be retried later), an error should be returned.
// Otherwise, it should return nil. The handler should handle any error logging as Processor
// implementations are not required to do so.
//
// Note that the task queue Processor is not expected to perform error logging, should the
// Handler encounter an error.
type Handler func(ctx context.Context, taskMsg *task.Message) error

// ProcessorCtor creates a new Processor configured with the provided Handler.
type ProcessorCtor func(handler Handler) (Processor, error)

// Processor processes messages from the task queue.
//
// Processors may also act as submitters.
type Processor interface {
	Submitter

	Shutdown()
}

// Submitter submits messages to the task queue.
type Submitter interface {
	Submit(ctx context.Context, msg *task.Message) error
}
