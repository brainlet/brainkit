package brainkit

import "github.com/brainlet/brainkit/internal/types"

// Role defines a named set of permissions.
type Role = types.Role

// BusPermissions controls which topics a role can publish/subscribe/emit.
type BusPermissions = types.BusPermissions

// TopicFilter uses glob patterns with deny-before-allow evaluation.
type TopicFilter = types.TopicFilter

// CommandPermissions controls which catalog commands a role can invoke.
type CommandPermissions = types.CommandPermissions

// RegistrationPermissions controls resource creation.
type RegistrationPermissions = types.RegistrationPermissions

// Built-in role presets.
var (
	RoleAdmin    = types.RoleAdmin
	RoleService  = types.RoleService
	RoleGateway  = types.RoleGateway
	RoleObserver = types.RoleObserver
)
