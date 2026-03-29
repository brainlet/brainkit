package rbac

import "strings"

// Allows checks if a topic is permitted by this filter.
// Evaluation order: deny checked first (explicit deny wins), then allow.
// If neither matches → denied by default.
func (f TopicFilter) Allows(topic string) bool {
	// Empty filter = nothing allowed
	if len(f.Allow) == 0 && len(f.Deny) == 0 {
		return false
	}

	// Check deny first — explicit deny always wins
	for _, pattern := range f.Deny {
		if matchGlob(pattern, topic) {
			return false
		}
	}

	// Check allow
	for _, pattern := range f.Allow {
		if matchGlob(pattern, topic) {
			return true
		}
	}

	// Neither matched → denied
	return false
}

// AllowsCommand checks if a command is permitted.
func (c CommandPermissions) AllowsCommand(command string) bool {
	if len(c.Allow) == 0 && len(c.Deny) == 0 {
		return false
	}

	// Check deny first
	for _, d := range c.Deny {
		if d == command || d == "*" {
			return false
		}
	}

	// Check allow
	for _, a := range c.Allow {
		if a == command || a == "*" {
			return true
		}
	}

	return false
}

// matchGlob matches a topic against a pattern.
// "*" matches everything. "prefix.*" matches any topic starting with "prefix.".
// Exact match is checked first.
func matchGlob(pattern, topic string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == topic {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(topic, prefix)
	}
	return false
}
