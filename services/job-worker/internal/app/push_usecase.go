package app

import "context"

type Claimer interface {
	ClaimAndEnqueue(ctx context.Context, execution_id string) error
}

type PushUsecase struct {
	claimer Claimer
}

func (u *PushUsecase) ClaimAndEnqueue(ctx context.Context, execution_id string) error {
	return u.claimer.ClaimAndEnqueue(ctx, execution_id)
}

func NewPushUsecase(c Claimer) *PushUsecase {
	return &PushUsecase{
		claimer: c,
	}
}
