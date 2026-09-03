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

type ExecutionPublisher struct {
	publisher *pubsub.Publisher
}

var _ eventing.ExecutionPublisher = (*ExecutionPublisher)(nil)

func (p *ExecutionPublisher) Publish(ctx context.Context, executionId domain.ExecutionId) error {
	payload, err := proto.Marshal(&jobv1.ExecutionRequested{
		ExecutionId: executionId.String(),
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

func (p *ExecutionPublisher) Close() {
	p.publisher.Stop()
}

func NewExecutionPublisher(client *pubsub.Client, topicID string) *ExecutionPublisher {
	return &ExecutionPublisher{
		publisher: client.Publisher(topicID),
	}
}
