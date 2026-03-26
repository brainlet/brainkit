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
