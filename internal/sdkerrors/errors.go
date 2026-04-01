package sdkerrors

import "fmt"

// BrainkitError is the interface all brainkit errors implement.
// Code() returns a machine-readable error code (UPPER_SNAKE_CASE).
// Details() returns structured fields for programmatic inspection.
type BrainkitError interface {
	error
	Code() string
	Details() map[string]any
}

// ── Existing types (now with Code + Details) ─────────────────────────────────

// NotFoundError is returned when a named resource does not exist.
// Resource is one of: "tool", "agent", "shard", "module", "storage", "pool", "peer", "mcp-server".
type NotFoundError struct {
	Resource string
	Name     string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %q not found", e.Resource, e.Name)
}
func (e *NotFoundError) Code() string { return "NOT_FOUND" }
func (e *NotFoundError) Details() map[string]any {
	return map[string]any{"resource": e.Resource, "name": e.Name}
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
func (e *AlreadyExistsError) Code() string { return "ALREADY_EXISTS" }
func (e *AlreadyExistsError) Details() map[string]any {
	return map[string]any{"resource": e.Resource, "name": e.Name, "hint": e.Hint}
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
func (e *ValidationError) Code() string { return "VALIDATION_ERROR" }
func (e *ValidationError) Details() map[string]any {
	return map[string]any{"field": e.Field, "message": e.Message}
}

// TimeoutError is returned when an operation exceeds its deadline.
type TimeoutError struct {
	Operation string
}

func (e *TimeoutError) Error() string           { return fmt.Sprintf("timeout: %s", e.Operation) }
func (e *TimeoutError) Code() string            { return "TIMEOUT" }
func (e *TimeoutError) Details() map[string]any { return map[string]any{"operation": e.Operation} }

// WorkspaceEscapeError is returned when a file path escapes the workspace boundary.
type WorkspaceEscapeError struct {
	Path string
}

func (e *WorkspaceEscapeError) Error() string           { return fmt.Sprintf("path %q escapes workspace", e.Path) }
func (e *WorkspaceEscapeError) Code() string            { return "WORKSPACE_ESCAPE" }
func (e *WorkspaceEscapeError) Details() map[string]any { return map[string]any{"path": e.Path} }

// ── New types ────────────────────────────────────────────────────────────────

// PermissionDeniedError is returned when RBAC denies an operation.
type PermissionDeniedError struct {
	Source string // deployment source or plugin name
	Action string // "publish", "subscribe", "emit", "command", "register"
	Topic  string // topic or command that was denied
	Role   string // role that denied it
}

func (e *PermissionDeniedError) Error() string {
	return fmt.Sprintf("permission denied: %s cannot %s on %q (role: %s)", e.Source, e.Action, e.Topic, e.Role)
}
func (e *PermissionDeniedError) Code() string { return "PERMISSION_DENIED" }
func (e *PermissionDeniedError) Details() map[string]any {
	return map[string]any{"source": e.Source, "action": e.Action, "topic": e.Topic, "role": e.Role}
}

// RateLimitedError is returned when a rate limit is exceeded.
type RateLimitedError struct {
	Role  string
	Limit float64
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("rate limit exceeded for role %q (limit: %.0f req/s)", e.Role, e.Limit)
}
func (e *RateLimitedError) Code() string { return "RATE_LIMITED" }
func (e *RateLimitedError) Details() map[string]any {
	return map[string]any{"role": e.Role, "limit": e.Limit}
}

// NotConfiguredError is returned when a required feature is not configured.
type NotConfiguredError struct {
	Feature string // "rbac", "mcp", "discovery", "tracing", "secrets", "workspace"
}

func (e *NotConfiguredError) Error() string           { return fmt.Sprintf("%s not configured", e.Feature) }
func (e *NotConfiguredError) Code() string            { return "NOT_CONFIGURED" }
func (e *NotConfiguredError) Details() map[string]any { return map[string]any{"feature": e.Feature} }

// TransportError is returned when a Watermill transport operation fails.
type TransportError struct {
	Operation string
	Cause     error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("transport: %s: %v", e.Operation, e.Cause)
}
func (e *TransportError) Unwrap() error           { return e.Cause }
func (e *TransportError) Code() string            { return "TRANSPORT_ERROR" }
func (e *TransportError) Details() map[string]any { return map[string]any{"operation": e.Operation} }

// PersistenceError is returned when a persistence (KitStore) operation fails.
type PersistenceError struct {
	Operation string // "SaveDeployment", "LoadDeployments", "SaveSchedule", etc.
	Source    string // deployment source or resource ID
	Cause     error
}

func (e *PersistenceError) Error() string {
	if e.Source != "" {
		return fmt.Sprintf("persistence: %s %s: %v", e.Operation, e.Source, e.Cause)
	}
	return fmt.Sprintf("persistence: %s: %v", e.Operation, e.Cause)
}
func (e *PersistenceError) Unwrap() error { return e.Cause }
func (e *PersistenceError) Code() string  { return "PERSISTENCE_ERROR" }
func (e *PersistenceError) Details() map[string]any {
	return map[string]any{"operation": e.Operation, "source": e.Source}
}

// DeployError is returned when a .ts deployment fails.
type DeployError struct {
	Source string // .ts filename
	Phase  string // "transpile", "eval", "compartment"
	Cause  error
}

func (e *DeployError) Error() string {
	return fmt.Sprintf("deploy %s: %s: %v", e.Source, e.Phase, e.Cause)
}
func (e *DeployError) Unwrap() error   { return e.Cause }
func (e *DeployError) Code() string    { return "DEPLOY_ERROR" }
func (e *DeployError) Details() map[string]any {
	return map[string]any{"source": e.Source, "phase": e.Phase}
}

// BridgeError is returned when a Go↔JS bridge function fails.
type BridgeError struct {
	Function string // bridge function name, e.g. "secret_get", "__go_brainkit_request"
	Cause    error
}

func (e *BridgeError) Error() string           { return fmt.Sprintf("bridge %s: %v", e.Function, e.Cause) }
func (e *BridgeError) Unwrap() error           { return e.Cause }
func (e *BridgeError) Code() string            { return "BRIDGE_ERROR" }
func (e *BridgeError) Details() map[string]any { return map[string]any{"function": e.Function} }

// CompilerError is returned when the AssemblyScript compiler fails.
type CompilerError struct {
	Cause error
}

func (e *CompilerError) Error() string           { return fmt.Sprintf("compiler: %v", e.Cause) }
func (e *CompilerError) Unwrap() error           { return e.Cause }
func (e *CompilerError) Code() string            { return "COMPILER_ERROR" }
func (e *CompilerError) Details() map[string]any { return map[string]any{} }

// CycleDetectedError is returned when message cascading exceeds the maximum depth.
type CycleDetectedError struct {
	Depth int
}

func (e *CycleDetectedError) Error() string {
	return fmt.Sprintf("cycle detected: depth %d exceeds maximum", e.Depth)
}
func (e *CycleDetectedError) Code() string            { return "CYCLE_DETECTED" }
func (e *CycleDetectedError) Details() map[string]any { return map[string]any{"depth": e.Depth} }

// DecodeError is returned when a message payload can't be decoded.
type DecodeError struct {
	Topic string
	Cause error
}

func (e *DecodeError) Error() string           { return fmt.Sprintf("decode %s: %v", e.Topic, e.Cause) }
func (e *DecodeError) Unwrap() error           { return e.Cause }
func (e *DecodeError) Code() string            { return "DECODE_ERROR" }
func (e *DecodeError) Details() map[string]any { return map[string]any{"topic": e.Topic} }

// ReplyDeniedError is returned when a reply is rejected due to invalid/missing token.
type ReplyDeniedError struct {
	Source        string
	ReplyTo       string
	CorrelationID string
}

func (e *ReplyDeniedError) Error() string {
	return fmt.Sprintf("reply denied: %s cannot reply to %s (correlationId: %s)", e.Source, e.ReplyTo, e.CorrelationID)
}
func (e *ReplyDeniedError) Code() string { return "REPLY_DENIED" }
func (e *ReplyDeniedError) Details() map[string]any {
	return map[string]any{"source": e.Source, "replyTo": e.ReplyTo, "correlationId": e.CorrelationID}
}
