package types

import (
	"strings"
	"time"
)

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

// Allows checks if a topic is permitted by this filter.
func (f TopicFilter) Allows(topic string) bool {
	if len(f.Allow) == 0 && len(f.Deny) == 0 {
		return false
	}
	for _, pattern := range f.Deny {
		if matchGlob(pattern, topic) {
			return false
		}
	}
	for _, pattern := range f.Allow {
		if matchGlob(pattern, topic) {
			return true
		}
	}
	return false
}

// AllowsCommand checks if a command is permitted.
func (c CommandPermissions) AllowsCommand(command string) bool {
	if len(c.Allow) == 0 && len(c.Deny) == 0 {
		return false
	}
	for _, d := range c.Deny {
		if d == command || d == "*" {
			return false
		}
	}
	for _, a := range c.Allow {
		if a == command || a == "*" {
			return true
		}
	}
	return false
}

func matchGlob(pattern, topic string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == topic {
		return true
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return false
	}
	remaining := topic
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(remaining, part)
		if idx < 0 {
			return false
		}
		if i == 0 && idx != 0 {
			return false
		}
		remaining = remaining[idx+len(part):]
	}
	if parts[len(parts)-1] != "" {
		return len(remaining) == 0
	}
	return true
}
