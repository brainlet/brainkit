package sdk

import (
	"fmt"

	"github.com/brainlet/brainkit/internal/sdkerrors"
)

// ── Sentinel errors ──────────────────────────────────────────────────────────
// Use errors.Is(err, sdk.ErrXxx) to check.

// ErrNoReplyTo is returned when a message has no replyTo metadata.
// Happens when calling Reply/SendChunk on a fire-and-forget message.
var ErrNoReplyTo = fmt.Errorf("sdk: message has no replyTo metadata")

// ErrNotReplier is returned when the runtime does not implement the Replier interface.
var ErrNotReplier = fmt.Errorf("sdk: runtime does not support Reply (does not implement Replier)")

// ErrNotCrossNamespace is returned when the runtime does not support cross-Kit operations.
var ErrNotCrossNamespace = fmt.Errorf("sdk: runtime does not support cross-namespace operations")

// ── Typed errors ─────────────────────────────────────────────────────────────
// Use errors.As(err, &target) to inspect fields.
// Definitions live in internal/sdkerrors to avoid import cycles; re-exported here as type aliases.

// NotFoundError is returned when a named resource does not exist.
// Resource is one of: "tool", "agent", "shard", "module", "storage", "pool", "peer", "mcp-server".
type NotFoundError = sdkerrors.NotFoundError

// AlreadyExistsError is returned when creating a resource that already exists.
// Resource is one of: "deployment", "shard", "storage", "pool".
type AlreadyExistsError = sdkerrors.AlreadyExistsError

// ValidationError is returned when input fails validation.
type ValidationError = sdkerrors.ValidationError

// TimeoutError is returned when an operation exceeds its deadline.
type TimeoutError = sdkerrors.TimeoutError

// WorkspaceEscapeError is returned when a file path escapes the workspace boundary.
type WorkspaceEscapeError = sdkerrors.WorkspaceEscapeError

// BrainkitError is the interface all brainkit errors implement.
type BrainkitError = sdkerrors.BrainkitError

// PermissionDeniedError is returned when RBAC denies an operation.
type PermissionDeniedError = sdkerrors.PermissionDeniedError

// RateLimitedError is returned when a rate limit is exceeded.
type RateLimitedError = sdkerrors.RateLimitedError

// NotConfiguredError is returned when a required feature is not configured.
type NotConfiguredError = sdkerrors.NotConfiguredError

// TransportError is returned when a transport operation fails.
type TransportError = sdkerrors.TransportError

// PersistenceError is returned when a persistence operation fails.
type PersistenceError = sdkerrors.PersistenceError

// DeployError is returned when a .ts deployment fails.
type DeployError = sdkerrors.DeployError

// BridgeError is returned when a Go↔JS bridge function fails.
type BridgeError = sdkerrors.BridgeError

// CompilerError is returned when the AssemblyScript compiler fails.
type CompilerError = sdkerrors.CompilerError

// CycleDetectedError is returned when message cascading exceeds max depth.
type CycleDetectedError = sdkerrors.CycleDetectedError

// DecodeError is returned when a message payload can't be decoded.
type DecodeError = sdkerrors.DecodeError
