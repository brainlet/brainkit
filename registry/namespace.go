package registry

import "strings"

// ParseNamespace extracts namespace and short name from a full tool name.
// "plugin.postgres@1.0.0.db_query" → ("plugin.postgres@1.0.0", "db_query")
// "db_query" → ("", "db_query")
func ParseNamespace(fullName string) (namespace, shortName string) {
	idx := strings.LastIndex(fullName, ".")
	if idx < 0 {
		return "", fullName
	}
	return fullName[:idx], fullName[idx+1:]
}

// ResolutionOrder returns the namespace search order for a caller.
// Caller "agent.coder-1" → ["agent.coder-1", "user", "platform"]
// Plugin namespaces are checked as a fallback after these.
func ResolutionOrder(callerNamespace string) []string {
	order := []string{callerNamespace}
	if callerNamespace != "user" {
		order = append(order, "user")
	}
	if callerNamespace != "platform" {
		order = append(order, "platform")
	}
	return order
}
