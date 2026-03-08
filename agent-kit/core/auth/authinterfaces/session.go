// Ported from: packages/core/src/auth/interfaces/session.ts
package authinterfaces

import (
	"net/http"
	"time"
)

// Session represents an authenticated session.
type Session struct {
	// ID is the unique session identifier.
	ID string `json:"id"`
	// UserID is the user ID this session belongs to.
	UserID string `json:"userId"`
	// ExpiresAt is when the session expires.
	ExpiresAt time.Time `json:"expiresAt"`
	// CreatedAt is when the session was created.
	CreatedAt time.Time `json:"createdAt"`
	// Metadata holds additional session metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ISessionProvider is the provider interface for session management.
//
// Implement this interface to enable:
//   - Session creation on login
//   - Session validation on requests
//   - Session destruction on logout
//   - Session refresh for long-lived sessions
type ISessionProvider interface {
	// CreateSession creates a new session for a user.
	CreateSession(userID string, metadata map[string]any) (*Session, error)

	// ValidateSession validates a session and returns it if valid.
	// Returns nil if invalid/expired.
	ValidateSession(sessionID string) (*Session, error)

	// DestroySession destroys a session (logout).
	DestroySession(sessionID string) error

	// RefreshSession refreshes a session, extending its expiry.
	// Returns nil if invalid.
	RefreshSession(sessionID string) (*Session, error)

	// GetSessionIDFromRequest extracts a session ID from an incoming request.
	// Returns "" if not present.
	GetSessionIDFromRequest(r *http.Request) string

	// GetSessionHeaders creates response headers to set session cookie/token.
	GetSessionHeaders(session *Session) map[string]string

	// GetClearSessionHeaders creates response headers to clear session (for logout).
	GetClearSessionHeaders() map[string]string
}
