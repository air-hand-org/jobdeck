package app

import "context"

// OutboxDispatcher triggers pending event publication.
type OutboxDispatcher interface {
	Trigger(ctx context.Context) error
}
