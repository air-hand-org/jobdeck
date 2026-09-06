// Package pubsub publishes job execution events to Google Cloud Pub/Sub.
package pubsub

import (
	"context"
	"jobdeck/services/job-api/internal/domain"
	"jobdeck/services/job-api/internal/eventing"

	jobv1 "github.com/air-hand-org/api-contracts/packages/jobdeck/job/go/v1"

	"cloud.google.com/go/pubsub/v2"
	"github.com/cockroachdb/errors"
	"google.golang.org/protobuf/proto"
)

// ExecutionPublisher publishes execution-requested events.
type ExecutionPublisher struct {
	publisher *pubsub.Publisher
}

var _ eventing.ExecutionPublisher = (*ExecutionPublisher)(nil)

// Publish publishes an execution-requested event for executionID.
func (p *ExecutionPublisher) Publish(ctx context.Context, executionID domain.ExecutionId) error {
	payload, err := proto.Marshal(&jobv1.ExecutionRequested{
		ExecutionId: executionID.String(),
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal ExecutionRequested")
	}

	result := p.publisher.Publish(ctx, &pubsub.Message{
		Data: payload,
	})

	if _, err := result.Get(ctx); err != nil {
		return errors.Wrap(err, "failed to publish ExecutionRequested")
	}
	return nil
}

// Close stops the underlying Pub/Sub publisher.
func (p *ExecutionPublisher) Close() {
	p.publisher.Stop()
}

// NewExecutionPublisher creates an execution event publisher for topicID.
func NewExecutionPublisher(client *pubsub.Client, topicID string) *ExecutionPublisher {
	return &ExecutionPublisher{
		publisher: client.Publisher(topicID),
	}
}
