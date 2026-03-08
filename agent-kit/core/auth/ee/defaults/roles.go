// Ported from: packages/core/src/auth/ee/defaults/roles.ts
package defaults

import (
	"strings"

	"github.com/brainlet/brainkit/agent-kit/core/auth/ee/eeinterfaces"
)

// ---------------------------------------------------------------------------
// Default roles
// ---------------------------------------------------------------------------

// DefaultRoles defines the default role definitions for Studio.
//
// These roles provide a sensible starting point for most applications:
//   - owner: Full access to everything
//   - admin: Manage agents, workflows, and users (no delete)
//   - member: Execute agents and workflows, read-only settings
//   - viewer: Read-only access
//
// Permission patterns:
//   - "*" — Full access to everything
//   - "resource:*" — All actions on a specific resource
//   - "*:action" — An action across all resources (e.g., "*:read" for read-only)
var DefaultRoles = []eeinterfaces.RoleDefinition{
	{
		ID:          "owner",
		Name:        "Owner",
		Description: "Full access to all features and settings",
		Permissions: []eeinterfaces.PermissionPattern{"*"},
	},
	{
		ID:          "admin",
		Name:        "Admin",
		Description: "Manage agents, workflows, and team members",
		Permissions: []eeinterfaces.PermissionPattern{
			"*:read",
			"*:write",
			"*:execute",
			// Note: admins cannot delete resources
		},
	},
	{
		ID:          "member",
		Name:        "Member",
		Description: "Execute agents and workflows",
		Permissions: []eeinterfaces.PermissionPattern{
			"*:read",
			"*:execute",
		},
	},
	{
		ID:          "viewer",
		Name:        "Viewer",
		Description: "Read-only access",
		Permissions: []eeinterfaces.PermissionPattern{"*:read"},
	},
}

// ---------------------------------------------------------------------------
// Role lookup
// ---------------------------------------------------------------------------

// GetDefaultRole returns a role definition by ID from DefaultRoles.
// Returns nil if not found.
func GetDefaultRole(roleID string) *eeinterfaces.RoleDefinition {
	for i := range DefaultRoles {
		if DefaultRoles[i].ID == roleID {
			return &DefaultRoles[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Permission resolution
// ---------------------------------------------------------------------------

// ResolvePermissions resolves all permissions for a set of role IDs.
//
// Handles role inheritance and deduplication. If roles is nil,
// DefaultRoles is used.
func ResolvePermissions(roleIDs []string, roles []eeinterfaces.RoleDefinition) []string {
	if roles == nil {
		roles = DefaultRoles
	}

	permissions := make(map[string]struct{})
	visited := make(map[string]struct{})

	var resolveRole func(roleID string)
	resolveRole = func(roleID string) {
		if _, ok := visited[roleID]; ok {
			return
		}
		visited[roleID] = struct{}{}

		var role *eeinterfaces.RoleDefinition
		for i := range roles {
			if roles[i].ID == roleID {
				role = &roles[i]
				break
			}
		}
		if role == nil {
			return
		}

		for _, permission := range role.Permissions {
			permissions[permission] = struct{}{}
		}

		// Resolve inherited roles
		for _, inheritedRoleID := range role.Inherits {
			resolveRole(inheritedRoleID)
		}
	}

	for _, roleID := range roleIDs {
		resolveRole(roleID)
	}

	result := make([]string, 0, len(permissions))
	for p := range permissions {
		result = append(result, p)
	}
	return result
}

// ---------------------------------------------------------------------------
// Permission matching
// ---------------------------------------------------------------------------

// MatchesPermission checks if a user permission matches a required permission
// (including wildcard support).
//
// Permission format: {resource}:{action}[:{resource-id}]
//
// Examples:
//   - "*" matches everything
//   - "agents:*" matches "agents:read", "agents:read:my-agent"
//   - "*:read" matches "agents:read", "workflows:read" (action across all resources)
//   - "agents:read" matches "agents:read", "agents:read:my-agent"
//   - "agents:read:my-agent" matches only "agents:read:my-agent"
//   - "agents:*:my-agent" matches "agents:read:my-agent", "agents:write:my-agent"
func MatchesPermission(userPermission, requiredPermission string) bool {
	// Wildcard matches everything
	if userPermission == "*" {
		return true
	}

	grantedParts := strings.SplitN(userPermission, ":", 3)
	requiredParts := strings.SplitN(requiredPermission, ":", 3)

	// Must have at least resource:action
	if len(grantedParts) < 2 || len(requiredParts) < 2 {
		return userPermission == requiredPermission
	}

	grantedResource := grantedParts[0]
	grantedAction := grantedParts[1]
	grantedID := ""
	if len(grantedParts) > 2 {
		grantedID = grantedParts[2]
	}

	requiredResource := requiredParts[0]
	requiredAction := requiredParts[1]
	requiredID := ""
	if len(requiredParts) > 2 {
		requiredID = requiredParts[2]
	}

	hasGrantedID := len(grantedParts) > 2

	// Resource wildcard: "*:*" matches everything, "*:read" matches any resource with that action
	if grantedResource == "*" {
		// "*:*" is a full wildcard — matches everything
		if grantedAction == "*" {
			if !hasGrantedID {
				return true
			}
			return grantedID == requiredID
		}
		// Action must match for resource wildcards with specific action
		if grantedAction != requiredAction {
			return false
		}
		// If no granted ID, matches all instances
		if !hasGrantedID {
			return true
		}
		// *:read:my-id would match agents:read:my-id (unusual but consistent)
		return grantedID == requiredID
	}

	// Resource must match (for non-wildcard resources)
	if grantedResource != requiredResource {
		return false
	}

	// Action wildcard: "agents:*" matches any action
	if grantedAction == "*" {
		// If no granted ID, matches all resources
		// If granted ID specified (agents:*:my-agent), must match required ID
		if !hasGrantedID {
			return true
		}
		// agents:*:my-agent matches agents:read:my-agent but not agents:read:other
		return grantedID == requiredID
	}

	// Action must match
	if grantedAction != requiredAction {
		return false
	}

	// No resource ID in granted permission = access to all resources of this type
	// "agents:read" matches "agents:read" and "agents:read:specific-id"
	if !hasGrantedID {
		return true
	}

	// Both have resource IDs — must match exactly
	return grantedID == requiredID
}

// HasPermission checks if a user has a specific permission.
func HasPermission(userPermissions []string, requiredPermission string) bool {
	for _, p := range userPermissions {
		if MatchesPermission(p, requiredPermission) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Role mapping resolution
// ---------------------------------------------------------------------------

// ResolvePermissionsFromMapping resolves permissions from user roles using a role mapping.
//
// This function translates provider-defined roles (from WorkOS, Okta, etc.)
// to Mastra permissions using a configurable mapping.
//
// Special key "_default" provides permissions for roles not explicitly mapped.
func ResolvePermissionsFromMapping(roles []string, mapping eeinterfaces.RoleMapping) []string {
	permissions := make(map[string]struct{})
	defaultPerms := mapping["_default"]

	for _, role := range roles {
		rolePerms, ok := mapping[role]
		if ok {
			for _, perm := range rolePerms {
				permissions[perm] = struct{}{}
			}
		} else {
			// Apply default permissions for unmapped roles
			for _, perm := range defaultPerms {
				permissions[perm] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(permissions))
	for p := range permissions {
		result = append(result, p)
	}
	return result
}
