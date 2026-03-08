// Ported from: packages/core/src/auth/ee/defaults/rbac/static.ts
package rbac

import (
	"sort"
	"strings"
	"sync"

	"github.com/brainlet/brainkit/agent-kit/core/auth/ee/defaults"
	"github.com/brainlet/brainkit/agent-kit/core/auth/ee/eeinterfaces"
)

// ---------------------------------------------------------------------------
// StaticRBACProviderOptions
// ---------------------------------------------------------------------------

// StaticRBACProviderOptions holds options for StaticRBACProvider.
//
// Use ONE of the following approaches:
//   - Roles: Define role structures with permissions (Mastra's native role system)
//   - RoleMapping: Map provider roles directly to permissions (simpler for external providers)
type StaticRBACProviderOptions[T any] struct {
	// Roles are the role definitions (Mastra's native role system).
	// Mutually exclusive with RoleMapping.
	Roles []eeinterfaces.RoleDefinition

	// RoleMapping maps provider roles directly to permissions.
	// Use this when your identity provider has roles that need to be
	// mapped to Mastra permissions.
	// Mutually exclusive with Roles.
	RoleMapping eeinterfaces.RoleMapping

	// GetUserRoles is a function to get user's role IDs.
	GetUserRoles func(user T) ([]string, error)
}

// ---------------------------------------------------------------------------
// StaticRBACProvider
// ---------------------------------------------------------------------------

// StaticRBACProvider is a static RBAC provider with config-based roles.
//
// Supports two modes:
//  1. Role definitions: Use Mastra's native role system with structured roles
//  2. Role mapping: Directly map provider roles to permissions
type StaticRBACProvider[T any] struct {
	roles           []eeinterfaces.RoleDefinition
	roleMapping     eeinterfaces.RoleMapping
	getUserRolesFn  func(user T) ([]string, error)
	permissionCache sync.Map // map[string][]string
}

// NewStaticRBACProvider creates a new StaticRBACProvider.
func NewStaticRBACProvider[T any](opts StaticRBACProviderOptions[T]) *StaticRBACProvider[T] {
	return &StaticRBACProvider[T]{
		roles:          opts.Roles,
		roleMapping:    opts.RoleMapping,
		getUserRolesFn: opts.GetUserRoles,
	}
}

// GetRoleMapping returns the optional role mapping for translating provider roles
// to Mastra permissions. Returns nil if not configured.
func (p *StaticRBACProvider[T]) GetRoleMapping() eeinterfaces.RoleMapping {
	return p.roleMapping
}

// GetRoles gets all roles for a user.
func (p *StaticRBACProvider[T]) GetRoles(user T) ([]string, error) {
	return p.getUserRolesFn(user)
}

// HasRole checks if user has a specific role.
func (p *StaticRBACProvider[T]) HasRole(user T, role string) (bool, error) {
	roles, err := p.GetRoles(user)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r == role {
			return true, nil
		}
	}
	return false, nil
}

// GetPermissions gets all permissions for a user (resolved from roles).
func (p *StaticRBACProvider[T]) GetPermissions(user T) ([]string, error) {
	roleIDs, err := p.GetRoles(user)
	if err != nil {
		return nil, err
	}

	// Build cache key
	sorted := make([]string, len(roleIDs))
	copy(sorted, roleIDs)
	sort.Strings(sorted)
	cacheKey := strings.Join(sorted, ",")

	// Check cache
	if cached, ok := p.permissionCache.Load(cacheKey); ok {
		return cached.([]string), nil
	}

	// Resolve permissions based on mode
	var permissions []string
	if p.roleMapping != nil {
		// Role mapping mode: translate provider roles to permissions
		permissions = defaults.ResolvePermissionsFromMapping(roleIDs, p.roleMapping)
	} else if p.roles != nil {
		// Role definitions mode: use Mastra's native role system
		permissions = defaults.ResolvePermissions(roleIDs, p.roles)
	} else {
		// No roles or mapping configured
		permissions = []string{}
	}

	// Cache result
	p.permissionCache.Store(cacheKey, permissions)

	return permissions, nil
}

// HasPermission checks if user has a specific permission.
func (p *StaticRBACProvider[T]) HasPermission(user T, permission string) (bool, error) {
	permissions, err := p.GetPermissions(user)
	if err != nil {
		return false, err
	}
	return defaults.HasPermission(permissions, permission), nil
}

// HasAllPermissions checks if user has ALL of the specified permissions.
func (p *StaticRBACProvider[T]) HasAllPermissions(user T, permissions []string) (bool, error) {
	userPermissions, err := p.GetPermissions(user)
	if err != nil {
		return false, err
	}
	for _, required := range permissions {
		if !defaults.HasPermission(userPermissions, required) {
			return false, nil
		}
	}
	return true, nil
}

// HasAnyPermission checks if user has ANY of the specified permissions.
func (p *StaticRBACProvider[T]) HasAnyPermission(user T, permissions []string) (bool, error) {
	userPermissions, err := p.GetPermissions(user)
	if err != nil {
		return false, err
	}
	for _, required := range permissions {
		if defaults.HasPermission(userPermissions, required) {
			return true, nil
		}
	}
	return false, nil
}

// ClearCache clears the permission cache.
func (p *StaticRBACProvider[T]) ClearCache() {
	p.permissionCache = sync.Map{}
}

// GetRoleDefinitions returns all role definitions.
// Only available when using role definitions mode (not role mapping).
func (p *StaticRBACProvider[T]) GetRoleDefinitions() []eeinterfaces.RoleDefinition {
	if p.roles == nil {
		return []eeinterfaces.RoleDefinition{}
	}
	return p.roles
}

// GetRoleDefinition returns a specific role definition by ID.
// Only available when using role definitions mode (not role mapping).
func (p *StaticRBACProvider[T]) GetRoleDefinition(roleID string) *eeinterfaces.RoleDefinition {
	if p.roles == nil {
		return nil
	}
	for i := range p.roles {
		if p.roles[i].ID == roleID {
			return &p.roles[i]
		}
	}
	return nil
}

// Compile-time check that StaticRBACProvider implements IRBACProvider.
var _ eeinterfaces.IRBACProvider[any] = (*StaticRBACProvider[any])(nil)
