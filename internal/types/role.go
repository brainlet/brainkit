package types

import "time"

// Role defines a named set of permissions.
type Role struct {
	Name         string                  `json:"name"`
	Bus          BusPermissions          `json:"bus"`
	Commands     CommandPermissions      `json:"commands"`
	Registration RegistrationPermissions `json:"registration"`
}

// BusPermissions controls which topics a role can publish/subscribe/emit.
type BusPermissions struct {
	Publish   TopicFilter `json:"publish"`
	Subscribe TopicFilter `json:"subscribe"`
	Emit      TopicFilter `json:"emit"`
}

// TopicFilter uses glob patterns with deny-before-allow evaluation.
type TopicFilter struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// CommandPermissions controls which catalog commands a role can invoke.
type CommandPermissions struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// RegistrationPermissions controls resource creation.
type RegistrationPermissions struct {
	Tools  bool `json:"tools"`
	Agents bool `json:"agents"`
}

// RoleAssignment records an explicit role assignment.
type RoleAssignment struct {
	Source     string    `json:"source"`
	Role       string    `json:"role"`
	AssignedAt time.Time `json:"assignedAt"`
}

// Four built-in role presets.

var RoleAdmin = Role{
	Name: "admin",
	Bus: BusPermissions{
		Publish:   TopicFilter{Allow: []string{"*"}},
		Subscribe: TopicFilter{Allow: []string{"*"}},
		Emit:      TopicFilter{Allow: []string{"*"}},
	},
	Commands:     CommandPermissions{Allow: []string{"*"}},
	Registration: RegistrationPermissions{Tools: true, Agents: true},
}

var RoleService = Role{
	Name: "service",
	Bus: BusPermissions{
		Publish:   TopicFilter{Allow: []string{"incoming.*", "events.*"}},
		Subscribe: TopicFilter{Allow: []string{"*.reply.*"}},
		Emit:      TopicFilter{Allow: []string{"events.*"}},
	},
	Commands: CommandPermissions{Allow: []string{
		"tools.call", "tools.list", "tools.resolve",
		"secrets.get",
	}},
	Registration: RegistrationPermissions{Tools: true, Agents: false},
}

var RoleGateway = Role{
	Name: "gateway",
	Bus: BusPermissions{
		Publish:   TopicFilter{Allow: []string{"incoming.*", "gateway.*"}},
		Subscribe: TopicFilter{Allow: []string{"*.reply.*"}},
		Emit:      TopicFilter{Allow: []string{"gateway.*"}},
	},
	Commands:     CommandPermissions{Allow: []string{}},
	Registration: RegistrationPermissions{},
}

var RoleObserver = Role{
	Name: "observer",
	Bus: BusPermissions{
		Subscribe: TopicFilter{Allow: []string{"*"}},
	},
	Commands: CommandPermissions{Allow: []string{
		"tools.list", "kit.list", "registry.list", "registry.has",
	}},
	Registration: RegistrationPermissions{},
}
