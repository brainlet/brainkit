// Ported from: packages/core/src/auth/ee/interfaces/acl.ts
package eeinterfaces

import "time"

// ResourceIdentifier identifies a resource.
type ResourceIdentifier struct {
	// Type is the resource type (e.g., "agent", "workflow", "thread").
	Type string `json:"type"`
	// ID is the resource ID.
	ID string `json:"id"`
}

// ACLSubject is the subject of an ACL grant (user or role).
type ACLSubject struct {
	// Type is the subject type: "user" or "role".
	Type string `json:"type"` // "user" | "role"
	// ID is the subject ID.
	ID string `json:"id"`
}

// ACLGrant is an access control grant.
type ACLGrant struct {
	// Subject is the subject of the grant (user or role).
	Subject ACLSubject `json:"subject"`
	// Resource is the resource the grant applies to.
	Resource ResourceIdentifier `json:"resource"`
	// Actions are the actions granted.
	Actions []string `json:"actions"`
	// GrantedAt is when the grant was created.
	GrantedAt time.Time `json:"grantedAt"`
	// GrantedBy is who created the grant.
	GrantedBy string `json:"grantedBy,omitempty"`
}

// Identifiable is a constraint for types that have an ID field.
type Identifiable interface {
	GetID() string
}

// IACLProvider is the provider interface for access control lists (read-only).
//
// Implement this interface to enable:
//   - Resource-level permission checks
//   - Filtered resource lists based on access
//   - ACL display in resource settings
type IACLProvider[T any] interface {
	// CanAccess checks if a user can perform an action on a resource.
	CanAccess(user T, resource ResourceIdentifier, action string) (bool, error)

	// ListAccessible gets a list of resource IDs the user can access.
	ListAccessible(user T, resourceType string, action string) ([]string, error)

	// FilterAccessible filters a slice of resources to only those the user can access.
	// The resources must have an ID accessible via GetID().
	FilterAccessible(user T, resourceIDs []string, resourceType string, action string) ([]string, error)
}

// IACLManager is the extended interface for managing ACLs (write operations).
// Implement this in addition to IACLProvider to enable ACL management.
type IACLManager[T any] interface {
	IACLProvider[T]

	// Grant grants access to a resource.
	Grant(subject ACLSubject, resource ResourceIdentifier, actions []string) error

	// Revoke revokes access to a resource.
	// If actions is nil, all actions are revoked.
	Revoke(subject ACLSubject, resource ResourceIdentifier, actions []string) error

	// ListGrants lists all grants for a resource.
	ListGrants(resource ResourceIdentifier) ([]ACLGrant, error)

	// ListGrantsForSubject lists all grants for a subject.
	ListGrantsForSubject(subject ACLSubject) ([]ACLGrant, error)
}
