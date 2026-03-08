// Ported from: packages/core/src/auth/interfaces/sso.ts
package authinterfaces

import "time"

// SSOLoginConfig is the configuration for rendering a login button.
type SSOLoginConfig struct {
	// Provider is the provider identifier (e.g., "mastra", "auth0", "okta").
	Provider string `json:"provider"`
	// Text is the button text (e.g., "Sign in with Mastra").
	Text string `json:"text"`
	// Icon is the optional icon URL.
	Icon string `json:"icon,omitempty"`
}

// SSOTokens holds OAuth tokens returned from an SSO callback.
type SSOTokens struct {
	// AccessToken is the access token for API calls.
	AccessToken string `json:"accessToken"`
	// RefreshToken is the refresh token for token renewal.
	RefreshToken string `json:"refreshToken,omitempty"`
	// IDToken is the ID token with user claims.
	IDToken string `json:"idToken,omitempty"`
	// ExpiresAt is the token expiration time.
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// SSOCallbackResult is the result of an SSO callback exchange.
type SSOCallbackResult struct {
	// User is the authenticated user.
	User User `json:"user"`
	// Tokens holds the OAuth tokens.
	Tokens SSOTokens `json:"tokens"`
	// Cookies are session cookies to set in the response.
	// Providers using encrypted cookie sessions (like AuthKit) should populate this.
	Cookies []string `json:"cookies,omitempty"`
}

// ISSOProvider is the provider interface for SSO authentication.
//
// Implement this interface to enable:
//   - SSO login button in Studio
//   - OAuth/OIDC redirect flows
//   - Token exchange on callback
type ISSOProvider interface {
	// GetLoginURL gets the URL to redirect the user to for login.
	GetLoginURL(redirectURI string, state string) string

	// HandleCallback handles the OAuth callback, exchanging a code for tokens and user.
	HandleCallback(code string, state string) (*SSOCallbackResult, error)

	// GetLogoutURL optionally gets the logout URL if the provider supports it.
	// Returns "" if not supported or no active session.
	GetLogoutURL(redirectURI string) (string, error)

	// GetLoginButtonConfig gets configuration for rendering the login button in UI.
	GetLoginButtonConfig() SSOLoginConfig

	// GetLoginCookies optionally gets cookies to set during login redirect.
	// Used by PKCE-enabled providers to store code verifier.
	// Returns nil if not applicable.
	GetLoginCookies(redirectURI string, state string) []string
}
