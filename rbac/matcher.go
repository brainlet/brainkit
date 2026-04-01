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

// matchGlob matches a topic against a pattern using segment-based wildcards.
// "*" alone matches everything. A "*" segment matches one or more topic segments.
// Examples:
//
//	"*"           matches everything
//	"incoming.*"  matches "incoming.foo", "incoming.foo.bar"
//	"*.reply.*"   matches "tools.call.reply.abc", "foo.reply.bar"
//	"events.*"    matches "events.deploy", "events.test.sub"
func matchGlob(pattern, topic string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == topic {
		return true
	}
	// Split pattern on "*" to get fixed segments that must appear in order.
	// E.g., "*.reply.*" → ["", ".reply.", ""]
	//        "incoming.*" → ["incoming.", ""]
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return false // no wildcards and not exact match
	}
	// Walk the topic string, matching each fixed part in sequence.
	// Empty parts (from leading/trailing *) match zero or more characters.
	remaining := topic
	for i, part := range parts {
		if part == "" {
			continue // wildcard consumes characters — handled by next fixed part
		}
		idx := strings.Index(remaining, part)
		if idx < 0 {
			return false // fixed part not found
		}
		if i == 0 && idx != 0 {
			return false // pattern doesn't start with * but topic has extra prefix
		}
		remaining = remaining[idx+len(part):]
	}
	// If the pattern ends with *, remaining can be anything (including empty).
	// If the pattern doesn't end with *, remaining must be empty.
	if parts[len(parts)-1] != "" {
		return len(remaining) == 0
	}
	return true
}
