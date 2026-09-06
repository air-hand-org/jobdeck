// Package domain contains auth domain types.
package domain

import "github.com/google/uuid"

// ActorId identifies an actor.
type ActorId = uuid.UUID //nolint:revive // Keep the domain name aligned with the protobuf contract.

// LocalAuthClient contains a local authentication client and its credentials.
type LocalAuthClient struct {
	ActorId    ActorId //nolint:revive // Keep the field name aligned with the protobuf contract.
	ClientId   string  //nolint:revive // Keep the field name aligned with the protobuf contract.
	SecretHash string
	Name       string
	Type       string // TODO ActorType
	Active     bool
}
