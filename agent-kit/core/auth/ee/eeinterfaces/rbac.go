// Ported from: packages/core/src/auth/ee/interfaces/rbac.ts
package eeinterfaces

// RoleDefinition defines a role with its permissions.
// Uses permission patterns derived from SERVER_ROUTES.
type RoleDefinition struct {
	// ID is the unique role identifier.
	ID string `json:"id"`
	// Name is the human-readable role name.
	Name string `json:"name"`
	// Description is the role description.
	Description string `json:"description,omitempty"`
	// Permissions are the permissions granted by this role.
	Permissions []PermissionPattern `json:"permissions"`
	// Inherits contains role IDs this role inherits from.
	Inherits []string `json:"inherits,omitempty"`
}

// RoleMapping is a role mapping configuration for translating provider roles to Mastra permissions.
//
// Use this when your identity provider (WorkOS, Okta, Azure AD, etc.) has its own
// roles that need to be translated to Mastra's permission model.
//
// Special keys:
//   - "_default": Permissions for roles not explicitly mapped
type RoleMapping = map[string][]PermissionPattern

// IRBACProvider is the provider interface for role-based access control (read-only).
//
// RBAC is designed to be separate from authentication.
// This allows users to mix auth providers with RBAC providers.
//
// Implement this interface to enable:
//   - Permission-based UI gating
//   - Role display in user menu
//   - Access control checks
type IRBACProvider[T any] interface {
	// GetRoleMapping returns the optional role mapping for translating provider roles
	// to Mastra permissions. Returns nil if not configured.
	GetRoleMapping() RoleMapping

	// GetRoles gets all roles for a user.
	GetRoles(user T) ([]string, error)

	// HasRole checks if user has a specific role.
	HasRole(user T, role string) (bool, error)

	// GetPermissions gets all permissions for a user (resolved from roles).
	GetPermissions(user T) ([]string, error)

	// HasPermission checks if user has a specific permission.
	HasPermission(user T, permission string) (bool, error)

	// HasAllPermissions checks if user has ALL of the specified permissions.
	HasAllPermissions(user T, permissions []string) (bool, error)

	// HasAnyPermission checks if user has ANY of the specified permissions.
	HasAnyPermission(user T, permissions []string) (bool, error)
}

// IRBACManager is the extended interface for managing roles (write operations).
// Implement this in addition to IRBACProvider to enable role management.
type IRBACManager[T any] interface {
	IRBACProvider[T]

	// AssignRole assigns a role to a user.
	AssignRole(userID string, roleID string) error

	// RemoveRole removes a role from a user.
	RemoveRole(userID string, roleID string) error

	// ListRoles lists all available roles.
	ListRoles() ([]RoleDefinition, error)

	// CreateRole optionally creates a new role.
	CreateRole(role RoleDefinition) error

	// DeleteRole optionally deletes a role.
	DeleteRole(roleID string) error
}
