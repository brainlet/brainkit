// Ported from: packages/core/src/server/composite-auth.ts
package server

import (
	"net/http"
	"sync"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// ----- Stub types for unported auth interfaces -----
// These mirror the interfaces from packages/core/src/auth/interfaces/.
// TODO: Port these to their own package when the auth package is ported.

// User is the base user type for authentication.
type User struct {
	ID        string `json:"id"`
	Email     string `json:"email,omitempty"`
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

// Session represents an authenticated session.
type Session struct {
	ID        string         `json:"id"`
	UserID    string         `json:"userId"`
	ExpiresAt int64          `json:"expiresAt"` // Unix timestamp
	CreatedAt int64          `json:"createdAt"` // Unix timestamp
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// SSOLoginConfig is the configuration for rendering a login button.
type SSOLoginConfig struct {
	Provider string `json:"provider"`
	Text     string `json:"text"`
	Icon     string `json:"icon,omitempty"`
}

// SSOCallbackTokens holds the OAuth tokens from an SSO callback.
type SSOCallbackTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	IDToken      string `json:"idToken,omitempty"`
	ExpiresAt    int64  `json:"expiresAt,omitempty"` // Unix timestamp
}

// SSOCallbackResult is the result of an SSO callback exchange.
type SSOCallbackResult struct {
	User    User              `json:"user"`
	Tokens  SSOCallbackTokens `json:"tokens"`
	Cookies []string          `json:"cookies,omitempty"`
}

// ISSOProvider is the interface for SSO authentication providers.
type ISSOProvider interface {
	GetLoginURL(redirectURI string, state string) string
	HandleCallback(code string, state string) (*SSOCallbackResult, error)
	GetLoginButtonConfig() SSOLoginConfig
	// Optional methods — check with type assertion.
	// GetLogoutURL(redirectURI string, r *http.Request) (string, error)
	// GetLoginCookies(redirectURI string, state string) []string
	// SetCallbackCookieHeader(cookieHeader string)
}

// ISessionProvider is the interface for session management providers.
type ISessionProvider interface {
	CreateSession(userID string, metadata map[string]any) (*Session, error)
	ValidateSession(sessionID string) (*Session, error)
	DestroySession(sessionID string) error
	RefreshSession(sessionID string) (*Session, error)
	GetSessionIDFromRequest(r *http.Request) string
	GetSessionHeaders(session *Session) map[string]string
	GetClearSessionHeaders() map[string]string
}

// IUserProvider is the interface for user awareness providers.
type IUserProvider interface {
	GetCurrentUser(r *http.Request) (*User, error)
	GetUser(userID string) (*User, error)
}

// ----- Optional interface extensions (checked via type assertion) -----

// SSOLogoutProvider is optionally implemented by ISSOProvider for logout support.
type SSOLogoutProvider interface {
	GetLogoutURL(redirectURI string, r *http.Request) (string, error)
}

// SSOLoginCookiesProvider is optionally implemented by ISSOProvider for PKCE cookie support.
type SSOLoginCookiesProvider interface {
	GetLoginCookies(redirectURI string, state string) []string
}

// SSOCallbackCookieProvider is optionally implemented by ISSOProvider for callback cookie forwarding.
type SSOCallbackCookieProvider interface {
	SetCallbackCookieHeader(cookieHeader string)
}

// ----- Type guards -----

// isSSOProvider checks if a provider implements ISSOProvider.
func isSSOProvider(p MastraAuthProvider) (ISSOProvider, bool) {
	sso, ok := p.(ISSOProvider)
	return sso, ok
}

// isSessionProvider checks if a provider implements ISessionProvider.
func isSessionProvider(p MastraAuthProvider) (ISessionProvider, bool) {
	sp, ok := p.(ISessionProvider)
	return sp, ok
}

// isUserProvider checks if a provider implements IUserProvider.
func isUserProvider(p MastraAuthProvider) (IUserProvider, bool) {
	up, ok := p.(IUserProvider)
	return up, ok
}

// ----- CompositeAuth -----

// CompositeAuth composes multiple MastraAuthProvider instances into a single provider.
// It delegates to the first provider that succeeds for authentication, and implements
// ISSOProvider, ISessionProvider, and IUserProvider by finding the first provider
// that implements each interface.
type CompositeAuth struct {
	*MastraAuthProviderBase
	providers []MastraAuthProvider
}

// Compile-time interface assertions.
var (
	_ MastraAuthProvider = (*CompositeAuth)(nil)
	_ ISSOProvider       = (*CompositeAuth)(nil)
	_ ISessionProvider   = (*CompositeAuth)(nil)
	_ IUserProvider      = (*CompositeAuth)(nil)
)

// NewCompositeAuth creates a new CompositeAuth from multiple providers.
func NewCompositeAuth(providers []MastraAuthProvider) *CompositeAuth {
	// Combine public and protected paths from all providers.
	var combinedPublic []PathRule
	var combinedProtected []PathRule

	for _, p := range providers {
		combinedPublic = append(combinedPublic, p.GetPublic()...)
		combinedProtected = append(combinedProtected, p.GetProtected()...)
	}

	base := NewMastraAuthProviderBase(&MastraAuthProviderOptions{
		Public:    combinedPublic,
		Protected: combinedProtected,
	})

	return &CompositeAuth{
		MastraAuthProviderBase: base,
		providers:              providers,
	}
}

// findSSOProvider returns the first provider implementing ISSOProvider.
func (c *CompositeAuth) findSSOProvider() (ISSOProvider, bool) {
	for _, p := range c.providers {
		if sso, ok := isSSOProvider(p); ok {
			return sso, true
		}
	}
	return nil, false
}

// findSessionProvider returns the first provider implementing ISessionProvider.
func (c *CompositeAuth) findSessionProvider() (ISessionProvider, bool) {
	for _, p := range c.providers {
		if sp, ok := isSessionProvider(p); ok {
			return sp, true
		}
	}
	return nil, false
}

// findUserProvider returns the first provider implementing IUserProvider.
func (c *CompositeAuth) findUserProvider() (IUserProvider, bool) {
	for _, p := range c.providers {
		if up, ok := isUserProvider(p); ok {
			return up, true
		}
	}
	return nil, false
}

// ============================================================================
// License Exemption Markers
// ============================================================================

// MastraCloudAuthMarker is implemented by providers exempt from license requirement.
type MastraCloudAuthMarker interface {
	IsMastraCloudAuth() bool
}

// SimpleAuthMarker is implemented by providers exempt from license requirement.
type SimpleAuthMarker interface {
	IsSimpleAuth() bool
}

// IsMastraCloudAuth returns true if any provider is MastraCloudAuth (exempt from license requirement).
func (c *CompositeAuth) IsMastraCloudAuth() bool {
	for _, p := range c.providers {
		if m, ok := p.(MastraCloudAuthMarker); ok && m.IsMastraCloudAuth() {
			return true
		}
	}
	return false
}

// IsSimpleAuth returns true if any provider is SimpleAuth (exempt from license requirement).
func (c *CompositeAuth) IsSimpleAuth() bool {
	for _, p := range c.providers {
		if m, ok := p.(SimpleAuthMarker); ok && m.IsSimpleAuth() {
			return true
		}
	}
	return false
}

// ============================================================================
// MastraAuthProvider Implementation
// ============================================================================

// AuthenticateToken tries each provider in order until one authenticates the token.
func (c *CompositeAuth) AuthenticateToken(token string, r *http.Request) (any, error) {
	for _, provider := range c.providers {
		user, err := provider.AuthenticateToken(token, r)
		if err != nil {
			// Ignore error, try next provider.
			continue
		}
		if user != nil {
			return user, nil
		}
	}
	return nil, nil
}

// AuthorizeUser tries each provider in order until one authorizes the user.
func (c *CompositeAuth) AuthorizeUser(user any, r *http.Request) (bool, error) {
	for _, provider := range c.providers {
		authorized, err := provider.AuthorizeUser(user, r)
		if err != nil {
			continue
		}
		if authorized {
			return true, nil
		}
	}
	return false, nil
}

// Logger returns the logger from the base provider.
func (c *CompositeAuth) Logger() logger.IMastraLogger {
	return c.MastraAuthProviderBase.Logger()
}

// ============================================================================
// ISSOProvider Implementation
// ============================================================================

// SetCallbackCookieHeader forwards cookie header to SSO provider for PKCE validation.
func (c *CompositeAuth) SetCallbackCookieHeader(cookieHeader string) {
	sso, ok := c.findSSOProvider()
	if !ok {
		return
	}
	if ccp, ok := sso.(SSOCallbackCookieProvider); ok {
		ccp.SetCallbackCookieHeader(cookieHeader)
	}
}

// GetLoginURL returns the login URL from the first SSO provider.
func (c *CompositeAuth) GetLoginURL(redirectURI string, state string) string {
	sso, ok := c.findSSOProvider()
	if !ok {
		panic("no SSO provider configured in CompositeAuth")
	}
	return sso.GetLoginURL(redirectURI, state)
}

// GetLoginCookies returns login cookies from the first SSO provider.
func (c *CompositeAuth) GetLoginCookies(redirectURI string, state string) []string {
	sso, ok := c.findSSOProvider()
	if !ok {
		return nil
	}
	if lcp, ok := sso.(SSOLoginCookiesProvider); ok {
		return lcp.GetLoginCookies(redirectURI, state)
	}
	return nil
}

// HandleCallback exchanges an authorization code for tokens via the first SSO provider.
func (c *CompositeAuth) HandleCallback(code string, state string) (*SSOCallbackResult, error) {
	sso, ok := c.findSSOProvider()
	if !ok {
		panic("no SSO provider configured in CompositeAuth")
	}
	return sso.HandleCallback(code, state)
}

// GetLoginButtonConfig returns the login button configuration from the first SSO provider.
func (c *CompositeAuth) GetLoginButtonConfig() SSOLoginConfig {
	sso, ok := c.findSSOProvider()
	if !ok {
		return SSOLoginConfig{Provider: "unknown", Text: "Sign in"}
	}
	return sso.GetLoginButtonConfig()
}

// GetLogoutURL tries each SSO provider until one returns a logout URL.
func (c *CompositeAuth) GetLogoutURL(redirectURI string, r *http.Request) (string, error) {
	for _, provider := range c.providers {
		sso, ok := isSSOProvider(provider)
		if !ok {
			continue
		}
		logout, ok := sso.(SSOLogoutProvider)
		if !ok {
			continue
		}
		url, err := logout.GetLogoutURL(redirectURI, r)
		if err != nil {
			// Try next provider.
			continue
		}
		if url != "" {
			return url, nil
		}
	}
	return "", nil
}

// ============================================================================
// ISessionProvider Implementation
// ============================================================================

// CreateSession creates a new session via the first session provider.
func (c *CompositeAuth) CreateSession(userID string, metadata map[string]any) (*Session, error) {
	sp, ok := c.findSessionProvider()
	if !ok {
		panic("no session provider configured in CompositeAuth")
	}
	return sp.CreateSession(userID, metadata)
}

// ValidateSession tries each session provider until one validates the session.
func (c *CompositeAuth) ValidateSession(sessionID string) (*Session, error) {
	for _, provider := range c.providers {
		sp, ok := isSessionProvider(provider)
		if !ok {
			continue
		}
		session, err := sp.ValidateSession(sessionID)
		if err != nil {
			// Try next provider.
			continue
		}
		if session != nil {
			return session, nil
		}
	}
	return nil, nil
}

// DestroySession destroys the session on ALL providers.
// A user may have sessions in multiple stores.
func (c *CompositeAuth) DestroySession(sessionID string) error {
	var wg sync.WaitGroup

	for _, provider := range c.providers {
		sp, ok := isSessionProvider(provider)
		if !ok {
			continue
		}
		wg.Add(1)
		go func(sp ISessionProvider) {
			defer wg.Done()
			// Ignore errors; session may not exist in this provider.
			_ = sp.DestroySession(sessionID)
		}(sp)
	}

	wg.Wait()
	return nil
}

// RefreshSession tries each session provider until one refreshes the session.
func (c *CompositeAuth) RefreshSession(sessionID string) (*Session, error) {
	for _, provider := range c.providers {
		sp, ok := isSessionProvider(provider)
		if !ok {
			continue
		}
		session, err := sp.RefreshSession(sessionID)
		if err != nil {
			// Try next provider.
			continue
		}
		if session != nil {
			return session, nil
		}
	}
	return nil, nil
}

// GetSessionIDFromRequest tries each session provider until one finds a session ID.
func (c *CompositeAuth) GetSessionIDFromRequest(r *http.Request) string {
	for _, provider := range c.providers {
		sp, ok := isSessionProvider(provider)
		if !ok {
			continue
		}
		sessionID := sp.GetSessionIDFromRequest(r)
		if sessionID != "" {
			return sessionID
		}
	}
	return ""
}

// GetSessionHeaders returns session headers from the first session provider.
// Intentionally uses only the first session provider: a session is created by one
// provider, so we only set its cookie.
func (c *CompositeAuth) GetSessionHeaders(session *Session) map[string]string {
	sp, ok := c.findSessionProvider()
	if !ok {
		return map[string]string{}
	}
	return sp.GetSessionHeaders(session)
}

// GetClearSessionHeaders merges clear headers from ALL providers to ensure
// no stale session cookies remain.
func (c *CompositeAuth) GetClearSessionHeaders() map[string]string {
	headers := map[string]string{}
	for _, provider := range c.providers {
		sp, ok := isSessionProvider(provider)
		if !ok {
			continue
		}
		for k, v := range sp.GetClearSessionHeaders() {
			headers[k] = v
		}
	}
	return headers
}

// ============================================================================
// IUserProvider Implementation
// ============================================================================

// GetCurrentUser tries each user provider until one returns a user.
func (c *CompositeAuth) GetCurrentUser(r *http.Request) (*User, error) {
	for _, provider := range c.providers {
		up, ok := isUserProvider(provider)
		if !ok {
			continue
		}
		user, err := up.GetCurrentUser(r)
		if err != nil {
			// Try next provider.
			continue
		}
		if user != nil {
			return user, nil
		}
	}
	return nil, nil
}

// GetUser tries each user provider until one returns a user by ID.
func (c *CompositeAuth) GetUser(userID string) (*User, error) {
	for _, provider := range c.providers {
		up, ok := isUserProvider(provider)
		if !ok {
			continue
		}
		user, err := up.GetUser(userID)
		if err != nil {
			// Try next provider.
			continue
		}
		if user != nil {
			return user, nil
		}
	}
	return nil, nil
}
