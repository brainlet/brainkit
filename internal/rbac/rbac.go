package rbac

import (
	"fmt"
	"strings"
	"github.com/brainlet/brainkit/internal/syncx"
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
	Role      string    `json:"role"`
	AssignedAt time.Time `json:"assignedAt"`
}

// Manager manages roles and role assignments.
type Manager struct {
	mu          syncx.RWMutex
	roles       map[string]*Role
	assignments map[string]RoleAssignment // source → assignment
	defaultRole string
}

// NewManager creates an RBAC manager with the given roles and default role.
// If no roles provided, the four built-in presets are used.
// If defaultRole is empty, "service" is used.
func NewManager(roles map[string]Role, defaultRole string) *Manager {
	if defaultRole == "" {
		defaultRole = "service"
	}
	m := &Manager{
		roles:       make(map[string]*Role),
		assignments: make(map[string]RoleAssignment),
		defaultRole: defaultRole,
	}

	// Register built-in presets first
	for _, preset := range []Role{RoleAdmin, RoleService, RoleGateway, RoleObserver} {
		p := preset
		m.roles[preset.Name] = &p
	}

	// Override with user-provided roles
	for name, role := range roles {
		r := role
		r.Name = name
		m.roles[name] = &r
	}

	return m
}

// RoleForSource returns the role for a .ts deployment source.
// Falls back to defaultRole if no explicit assignment.
func (m *Manager) RoleForSource(source string) *Role {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if assignment, ok := m.assignments[source]; ok {
		if role, ok := m.roles[assignment.Role]; ok {
			return role
		}
	}

	if role, ok := m.roles[m.defaultRole]; ok {
		return role
	}
	return m.roles["service"]
}

// RoleForPlugin returns the role for a plugin.
func (m *Manager) RoleForPlugin(pluginName string) *Role {
	return m.RoleForSource(pluginName)
}

// Assign sets the role for a source/plugin. Takes effect on next bridge call.
func (m *Manager) Assign(source, roleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.roles[roleName]; !ok {
		return fmt.Errorf("rbac: role %q not defined", roleName)
	}

	m.assignments[source] = RoleAssignment{
		Source:     source,
		Role:       roleName,
		AssignedAt: time.Now(),
	}
	return nil
}

// Revoke removes an explicit assignment. Falls back to defaultRole.
func (m *Manager) Revoke(source string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.assignments, source)
}

// ListAssignments returns all explicit role assignments.
func (m *Manager) ListAssignments() []RoleAssignment {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]RoleAssignment, 0, len(m.assignments))
	for _, a := range m.assignments {
		result = append(result, a)
	}
	return result
}

// ListRoles returns all defined roles.
func (m *Manager) ListRoles() []Role {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Role, 0, len(m.roles))
	for _, r := range m.roles {
		result = append(result, *r)
	}
	return result
}

// IsOwnMailbox checks if a topic belongs to a deployment's own namespace.
// Own mailbox is ALWAYS accessible regardless of role.
func IsOwnMailbox(source, topic string) bool {
	if source == "" {
		return false
	}
	name := strings.TrimSuffix(source, ".ts")
	name = strings.ReplaceAll(name, "/", ".")
	prefix := "ts." + name + "."
	return strings.HasPrefix(topic, prefix)
}
