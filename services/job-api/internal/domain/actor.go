package domain

import (
	"slices"

	"github.com/google/uuid"
	"github.com/samber/lo"
)

type ActorType string

const (
	ActorTypeUser    ActorType = "USER"
	ActorTypeService ActorType = "SERVICE"
)

type Capability string

const (
	CapJobRead     Capability = "job.read"
	CapJobWrite    Capability = "job.write"
	CapQueueClaim  Capability = "queue.claim"
	CapQueueReport Capability = "queue.report"
)

type Role struct {
	Caps []Capability `json:"caps"`
}

func (r Role) HasCapability(cap Capability) bool {
	return lo.Contains(r.Caps, cap)
}

type RoleName string

const (
	RoleUser   RoleName = "USER"
	RoleWorker RoleName = "WORKER"
)

type RoleDefinition struct {
	ActorType ActorType    `json:"actor_type"`
	Caps      []Capability `json:"caps"`
}

var roleDefinitions = map[RoleName]RoleDefinition{
	RoleUser: {
		ActorType: ActorTypeUser,
		Caps: []Capability{
			CapJobRead,
			CapJobWrite,
		},
	},
	RoleWorker: {
		ActorType: ActorTypeService,
		Caps: []Capability{
			CapQueueClaim,
			CapQueueReport,
		},
	},
}

func roleFromName(actor_type ActorType, name RoleName) (Role, error) {
	// if user doesnt has role, set role fallbackly
	if actor_type == ActorTypeUser && len(name) == 0 {
		name = RoleUser
	}

	if len(name) == 0 {
		return Role{}, ErrActorNotHasRole
	}

	role_def, ok := roleDefinitions[name]
	if !ok {
		return Role{}, ErrActorInvalidRole
	}
	if role_def.ActorType != actor_type {
		return Role{}, ErrActorInvalidRole
	}

	return Role{
		Caps: slices.Clone(role_def.Caps),
	}, nil
}

type ActorId = uuid.UUID

type AuthenticatedIdentity struct {
	ID   ActorId   `json:"id"`
	Type ActorType `json:"type"`
}

type Actor struct {
	ID     ActorId   `json:"id"`
	Active bool      `json:"active"`
	Type   ActorType `json:"type"`
	Role   Role      `json:"role"`
}

func NewActor(actor_id ActorId, active bool, actor_type ActorType, role_name RoleName) (Actor, error) {
	role, err := roleFromName(actor_type, role_name)
	if err != nil {
		return Actor{}, err
	}

	return Actor{
		ID:     actor_id,
		Active: active,
		Type:   actor_type,
		Role:   role,
	}, nil
}
