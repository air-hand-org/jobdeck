package domain

import (
	"slices"

	"github.com/google/uuid"
	"github.com/samber/lo"
)

// ActorType identifies the kind of actor.
type ActorType string

const (
	// ActorTypeUser identifies a human user.
	ActorTypeUser ActorType = "USER"
	// ActorTypeService identifies a service account.
	ActorTypeService ActorType = "SERVICE"
)

// Capability identifies an authorized operation.
type Capability string

const (
	// CapJobRead permits reading jobs.
	CapJobRead Capability = "job.read"
	// CapJobWrite permits modifying jobs.
	CapJobWrite Capability = "job.write"
	// CapQueueClaim permits claiming executions.
	CapQueueClaim Capability = "queue.claim"
	// CapQueueReport permits reporting execution results.
	CapQueueReport Capability = "queue.report"
)

// Role contains the capabilities granted to an actor.
type Role struct {
	Caps []Capability `json:"caps"`
}

// HasCapability reports whether the role includes capability.
func (r Role) HasCapability(capability Capability) bool {
	return lo.Contains(r.Caps, capability)
}

// RoleName identifies a role definition.
type RoleName string

const (
	// RoleUser is the default user role.
	RoleUser RoleName = "USER"
	// RoleWorker is the worker role.
	RoleWorker RoleName = "WORKER"
)

// RoleDefinition maps an actor type to its capabilities.
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

func roleFromName(actorType ActorType, name RoleName) (Role, error) {
	// if user doesnt has role, set role fallbackly
	if actorType == ActorTypeUser && len(name) == 0 {
		name = RoleUser
	}

	if len(name) == 0 {
		return Role{}, ErrActorNotHasRole
	}

	roleDef, ok := roleDefinitions[name]
	if !ok {
		return Role{}, ErrActorInvalidRole
	}
	if roleDef.ActorType != actorType {
		return Role{}, ErrActorInvalidRole
	}

	return Role{
		Caps: slices.Clone(roleDef.Caps),
	}, nil
}

// ActorId is the UUID of an actor.
//
//nolint:revive // Retained for compatibility with the existing internal API.
type ActorId = uuid.UUID

// AuthenticatedIdentity is the identity extracted from a valid token.
type AuthenticatedIdentity struct {
	ID   ActorId   `json:"id"`
	Type ActorType `json:"type"`
}

// Actor is an authenticated principal and its permissions.
type Actor struct {
	ID     ActorId   `json:"id"`
	Active bool      `json:"active"`
	Type   ActorType `json:"type"`
	Role   Role      `json:"role"`
}

// NewActor creates an actor with the specified role.
func NewActor(actorID ActorId, active bool, actorType ActorType, roleName RoleName) (Actor, error) {
	role, err := roleFromName(actorType, roleName)
	if err != nil {
		return Actor{}, err
	}

	return Actor{
		ID:     actorID,
		Active: active,
		Type:   actorType,
		Role:   role,
	}, nil
}
