// Ported from: packages/core/src/auth/interfaces/user.ts
package authinterfaces

import "net/http"

// User is the base user type for authentication.
type User struct {
	// ID is the unique user identifier.
	ID string `json:"id"`
	// Email is the user email address.
	Email string `json:"email,omitempty"`
	// Name is the display name.
	Name string `json:"name,omitempty"`
	// AvatarURL is the avatar URL.
	AvatarURL string `json:"avatarUrl,omitempty"`
}

// IUserProvider is the provider interface for user awareness in Studio.
//
// Implement this interface to enable:
//   - Current user display in header
//   - User menu with profile info
//   - User context in API calls
type IUserProvider interface {
	// GetCurrentUser gets the current user from the request (session cookie, token, etc.)
	// Returns nil if not authenticated.
	GetCurrentUser(r *http.Request) (*User, error)

	// GetUser gets a user by ID.
	// Returns nil if not found.
	GetUser(userID string) (*User, error)

	// GetUserProfileURL optionally returns a URL to the user's profile page.
	// Implementations may return "" if not supported.
	GetUserProfileURL(user *User) string
}
