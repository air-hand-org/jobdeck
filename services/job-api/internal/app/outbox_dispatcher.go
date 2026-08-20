package app

import "context"

type OutboxDispatcher interface {
	Trigger(ctx context.Context) error
}
