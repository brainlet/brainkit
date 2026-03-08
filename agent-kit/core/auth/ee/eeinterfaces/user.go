// Ported from: packages/core/src/auth/ee/interfaces/user.ts
package eeinterfaces

import (
	"github.com/brainlet/brainkit/agent-kit/core/auth/authinterfaces"
)

// EEUser is the enterprise user type with additional metadata.
// Extends the base User type with fields commonly needed
// for RBAC, ACL, and organizational features.
type EEUser struct {
	authinterfaces.User
	// Metadata holds additional metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
}
