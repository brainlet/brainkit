package sdkerrors

import "fmt"

// NotFoundError is returned when a named resource does not exist.
// Resource is one of: "tool", "agent", "shard", "module", "storage", "pool", "peer", "mcp-server".
type NotFoundError struct {
	Resource string
	Name     string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %q not found", e.Resource, e.Name)
}

// AlreadyExistsError is returned when creating a resource that already exists.
// Resource is one of: "deployment", "shard", "storage", "pool".
type AlreadyExistsError struct {
	Resource string
	Name     string
	Hint     string // optional action hint, e.g. "use Redeploy"
}

func (e *AlreadyExistsError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s %q already exists (%s)", e.Resource, e.Name, e.Hint)
	}
	return fmt.Sprintf("%s %q already exists", e.Resource, e.Name)
}

// ValidationError is returned when input fails validation.
type ValidationError struct {
	Field   string // field or parameter name
	Message string // human-readable reason
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation: %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation: %s", e.Message)
}

// TimeoutError is returned when an operation exceeds its deadline.
type TimeoutError struct {
	Operation string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout: %s", e.Operation)
}

// WorkspaceEscapeError is returned when a file path escapes the workspace boundary.
type WorkspaceEscapeError struct {
	Path string
}

func (e *WorkspaceEscapeError) Error() string {
	return fmt.Sprintf("path %q escapes workspace", e.Path)
}
