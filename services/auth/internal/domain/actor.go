package domain

import "github.com/google/uuid"

type ActorId = uuid.UUID

type LocalAuthClient struct {
	ActorId    ActorId
	ClientId   string
	SecretHash string
	Name       string
	Type       string // TODO ActorType
	Active     bool
}
