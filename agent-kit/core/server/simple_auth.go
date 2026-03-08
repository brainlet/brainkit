// Ported from: packages/core/src/server/simple-auth.ts
package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// DefaultHeaders are the default headers checked for authentication tokens.
var DefaultHeaders = []string{"Authorization", "X-Playground-Access"}

// ----- Stub type for unported credentials interface -----

// CredentialsResult is the result of a successful credentials operation.
// TODO: Port from packages/core/src/auth/interfaces/credentials.ts
type CredentialsResult struct {
	// User is the authenticated user.
	User any
	// Token is an optional session token.
	Token string
	// Cookies are optional cookies to set on the response.
	Cookies []string
}

// ----- SimpleAuthOptions -----

// SimpleAuthOptions holds configuration for creating a SimpleAuth provider.
type SimpleAuthOptions struct {
	MastraAuthProviderOptions
	// Tokens maps token strings to their associated user values.
	Tokens map[string]any
	// Headers are additional headers to check for authentication.
	// Default: ["Authorization", "X-Playground-Access"]
	Headers []string
}

// ----- SimpleAuth -----

// SimpleAuth is a simple token-based authentication provider for development and testing.
// It maps static tokens to user objects.
//
// SimpleAuth is exempt from EE license requirement (marked via IsSimpleAuth).
type SimpleAuth struct {
	*MastraAuthProviderBase

	tokens   map[string]any
	headers  []string
	users    []any
	userByID map[string]any
}

// Compile-time interface assertions.
var (
	_ MastraAuthProvider = (*SimpleAuth)(nil)
	_ SimpleAuthMarker   = (*SimpleAuth)(nil)
	_ IUserProvider      = (*SimpleAuth)(nil)
)

// NewSimpleAuth creates a new SimpleAuth provider with the given options.
func NewSimpleAuth(opts SimpleAuthOptions) *SimpleAuth {
	base := NewMastraAuthProviderBase(&opts.MastraAuthProviderOptions)

	users := make([]any, 0, len(opts.Tokens))
	for _, u := range opts.Tokens {
		users = append(users, u)
	}

	// Build headers list: defaults + custom headers.
	headers := make([]string, len(DefaultHeaders))
	copy(headers, DefaultHeaders)
	headers = append(headers, opts.Headers...)

	// Build userByID map. Attempts to extract "id" field from user values.
	userByID := make(map[string]any, len(users))
	for _, u := range users {
		if m, ok := u.(map[string]any); ok {
			if id, ok := m["id"]; ok {
				userByID[fmt.Sprintf("%v", id)] = u
			}
		} else if user, ok := u.(*User); ok {
			userByID[user.ID] = u
		} else if user, ok := u.(User); ok {
			userByID[user.ID] = u
		}
	}

	return &SimpleAuth{
		MastraAuthProviderBase: base,
		tokens:                 opts.Tokens,
		headers:                headers,
		users:                  users,
		userByID:               userByID,
	}
}

// IsSimpleAuth returns true, marking this provider as exempt from EE license requirement.
func (s *SimpleAuth) IsSimpleAuth() bool {
	return true
}

// Logger returns the underlying logger.
func (s *SimpleAuth) Logger() logger.IMastraLogger {
	return s.MastraAuthProviderBase.Logger()
}

// AuthenticateToken authenticates a token by checking request headers and cookies.
func (s *SimpleAuth) AuthenticateToken(token string, r *http.Request) (any, error) {
	requestTokens := s.getTokensFromHeaders(token, r)

	for _, rt := range requestTokens {
		if user, ok := s.tokens[rt]; ok {
			return user, nil
		}
	}

	// Fall back to cookie-based authentication.
	return s.getUserFromCookie(r.Header.Get("Cookie")), nil
}

// AuthorizeUser checks if the user is in the known users list.
func (s *SimpleAuth) AuthorizeUser(user any, _ *http.Request) (bool, error) {
	for _, u := range s.users {
		if u == user {
			return true, nil
		}
	}
	return false, nil
}

// GetCurrentUser gets the current user from request headers or cookie.
func (s *SimpleAuth) GetCurrentUser(r *http.Request) (*User, error) {
	// Check headers first.
	for _, headerName := range s.headers {
		headerValue := r.Header.Get(headerName)
		if headerValue != "" {
			token := stripBearerPrefix(headerValue)
			if user, ok := s.tokens[token]; ok {
				if u, ok := user.(*User); ok {
					return u, nil
				}
				if u, ok := user.(User); ok {
					return &u, nil
				}
				// Return nil if user type doesn't match *User.
				return nil, nil
			}
		}
	}

	// Fall back to cookie.
	result := s.getUserFromCookie(r.Header.Get("Cookie"))
	if result != nil {
		if u, ok := result.(*User); ok {
			return u, nil
		}
		if u, ok := result.(User); ok {
			return &u, nil
		}
	}
	return nil, nil
}

// GetUser returns a user by ID.
func (s *SimpleAuth) GetUser(userID string) (*User, error) {
	u, ok := s.userByID[userID]
	if !ok {
		return nil, nil
	}
	if user, ok := u.(*User); ok {
		return user, nil
	}
	if user, ok := u.(User); ok {
		return &user, nil
	}
	return nil, nil
}

// SignIn authenticates with a token (passed as the password field).
// The email field is ignored - only the token matters.
func (s *SimpleAuth) SignIn(_ string, password string, _ *http.Request) (*CredentialsResult, error) {
	token := password
	user, ok := s.tokens[token]
	if !ok {
		return nil, fmt.Errorf("invalid token")
	}

	// Set cookie so the token persists across requests.
	cookie := fmt.Sprintf("mastra-token=%s; Path=/; HttpOnly; SameSite=Lax; Max-Age=86400", token)

	return &CredentialsResult{
		User:    user,
		Token:   token,
		Cookies: []string{cookie},
	}, nil
}

// SignUp is not supported with SimpleAuth.
func (s *SimpleAuth) SignUp() (*CredentialsResult, error) {
	return nil, fmt.Errorf("sign up is not supported with SimpleAuth. Use pre-configured tokens")
}

// IsSignUpEnabled returns false. SimpleAuth does not support sign-up.
func (s *SimpleAuth) IsSignUpEnabled() bool {
	return false
}

// GetClearSessionHeaders returns headers to clear the session cookie on logout.
func (s *SimpleAuth) GetClearSessionHeaders() map[string]string {
	return map[string]string{
		"Set-Cookie": "mastra-token=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0",
	}
}

// ----- Private helpers -----

// getUserFromCookie extracts a user from the mastra-token cookie.
func (s *SimpleAuth) getUserFromCookie(cookieHeader string) any {
	if cookieHeader == "" {
		return nil
	}

	cookies := strings.Split(cookieHeader, ";")
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)
		if strings.HasPrefix(cookie, "mastra-token=") {
			token := strings.TrimPrefix(cookie, "mastra-token=")
			if user, ok := s.tokens[token]; ok {
				return user
			}
		}
	}
	return nil
}

// stripBearerPrefix removes the "Bearer " prefix from a token string.
func stripBearerPrefix(token string) string {
	if strings.HasPrefix(token, "Bearer ") {
		return token[7:]
	}
	return token
}

// getTokensFromHeaders collects tokens from the initial token and all configured headers.
func (s *SimpleAuth) getTokensFromHeaders(token string, r *http.Request) []string {
	tokens := []string{token}
	for _, headerName := range s.headers {
		headerValue := r.Header.Get(headerName)
		if headerValue != "" {
			tokens = append(tokens, stripBearerPrefix(headerValue))
		}
	}
	return tokens
}
