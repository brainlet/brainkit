// Type aliases from internal/types — rbac implementation uses these.
package rbac

import "github.com/brainlet/brainkit/internal/types"

type Role = types.Role
type BusPermissions = types.BusPermissions
type TopicFilter = types.TopicFilter
type CommandPermissions = types.CommandPermissions
type RegistrationPermissions = types.RegistrationPermissions
type RoleAssignment = types.RoleAssignment

var (
	RoleAdmin    = types.RoleAdmin
	RoleService  = types.RoleService
	RoleGateway  = types.RoleGateway
	RoleObserver = types.RoleObserver
)
