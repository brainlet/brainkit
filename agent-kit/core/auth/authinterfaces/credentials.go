// Ported from: packages/core/src/auth/interfaces/credentials.ts
package authinterfaces

import "net/http"

// CredentialsResult is the result of a successful credentials operation.
type CredentialsResult struct {
	// User is the authenticated user.
	User User `json:"user"`
	// Token is an optional session token.
	Token string `json:"token,omitempty"`
	// Cookies are optional cookies to set on the response (e.g., session cookies).
	Cookies []string `json:"cookies,omitempty"`
}

// ICredentialsProvider is the provider interface for credentials-based authentication in Studio.
//
// Implement this interface to enable:
//   - Email/password sign-in
//   - Email/password sign-up
//   - Password reset (optional)
type ICredentialsProvider interface {
	// SignIn signs in with email and password.
	// Returns an error if credentials are invalid.
	SignIn(email string, password string, r *http.Request) (*CredentialsResult, error)

	// SignUp signs up with email and password.
	// The name parameter may be empty.
	// Returns an error if sign up fails (e.g., email already exists).
	SignUp(email string, password string, name string, r *http.Request) (*CredentialsResult, error)

	// RequestPasswordReset optionally requests a password reset.
	// Implementations that don't support this should return nil.
	RequestPasswordReset(email string) error

	// ResetPassword optionally resets a password with a token.
	// Implementations that don't support this should return nil.
	ResetPassword(token string, newPassword string) error

	// IsSignUpEnabled optionally checks if sign-up is enabled.
	// Defaults to true if not implemented (return true).
	IsSignUpEnabled() bool
}
