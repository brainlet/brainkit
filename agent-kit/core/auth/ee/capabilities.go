// Ported from: packages/core/src/auth/ee/capabilities.ts
package ee

import (
	"net/http"

	"github.com/brainlet/brainkit/agent-kit/core/auth/authinterfaces"
	"github.com/brainlet/brainkit/agent-kit/core/auth/ee/eeinterfaces"
	"github.com/brainlet/brainkit/agent-kit/core/server"
)

// MastraAuthProvider is wired to the real server.MastraAuthProvider interface.
// No circular dependency: server does not import auth/ee.
// The server package defines the full MastraAuthProvider interface with methods:
// AuthenticateToken, AuthorizeUser, GetProtected, GetPublic, Logger.
type MastraAuthProvider = server.MastraAuthProvider

// PublicAuthCapabilities is the public capabilities response (no authentication required).
// Contains just enough info to render the login page.
type PublicAuthCapabilities struct {
	// Enabled indicates whether auth is enabled.
	Enabled bool `json:"enabled"`
	// Login is the login configuration (nil if no auth or no SSO).
	Login *LoginConfig `json:"login"`
}

// LoginConfig is the login configuration within PublicAuthCapabilities.
type LoginConfig struct {
	// Type is the type of login available: "sso", "credentials", or "both".
	Type string `json:"type"`
	// SignUpEnabled indicates whether sign-up is enabled (defaults to true).
	SignUpEnabled *bool `json:"signUpEnabled,omitempty"`
	// SSO is the SSO configuration.
	SSO *SSOConfig `json:"sso,omitempty"`
}

// SSOConfig is the SSO configuration within LoginConfig.
type SSOConfig struct {
	// Provider is the provider name.
	Provider string `json:"provider"`
	// Text is the button text.
	Text string `json:"text"`
	// Icon is the icon URL.
	Icon string `json:"icon,omitempty"`
	// URL is the login URL.
	URL string `json:"url"`
}

// AuthenticatedUser is the user info for an authenticated response.
type AuthenticatedUser struct {
	// ID is the user ID.
	ID string `json:"id"`
	// Email is the user email.
	Email string `json:"email,omitempty"`
	// Name is the display name.
	Name string `json:"name,omitempty"`
	// AvatarURL is the avatar URL.
	AvatarURL string `json:"avatarUrl,omitempty"`
}

// CapabilityFlags indicates which EE features are available.
type CapabilityFlags struct {
	// User indicates IUserProvider is implemented and licensed.
	User bool `json:"user"`
	// Session indicates ISessionProvider is implemented and licensed.
	Session bool `json:"session"`
	// SSO indicates ISSOProvider is implemented and licensed.
	SSO bool `json:"sso"`
	// RBAC indicates IRBACProvider is implemented and licensed.
	RBAC bool `json:"rbac"`
	// ACL indicates IACLProvider is implemented and licensed.
	ACL bool `json:"acl"`
}

// UserAccess represents a user's access (roles and permissions).
type UserAccess struct {
	// Roles are the user's roles.
	Roles []string `json:"roles"`
	// Permissions are the user's resolved permissions.
	Permissions []string `json:"permissions"`
}

// AuthenticatedCapabilities is the authenticated capabilities response.
// Extends PublicAuthCapabilities with user context and feature flags.
type AuthenticatedCapabilities struct {
	PublicAuthCapabilities
	// User is the current authenticated user.
	User AuthenticatedUser `json:"user"`
	// Capabilities are the available EE capabilities.
	Capabilities CapabilityFlags `json:"capabilities"`
	// Access is the user's access (if RBAC available).
	Access *UserAccess `json:"access"`
}

// IsAuthenticated is a type check to determine if a capabilities response is authenticated.
// Returns the AuthenticatedCapabilities and true if authenticated, nil and false otherwise.
func IsAuthenticated(caps *PublicAuthCapabilities, authCaps *AuthenticatedCapabilities) bool {
	return authCaps != nil
}

// implementsUserProvider checks if auth implements IUserProvider.
func implementsUserProvider(auth MastraAuthProvider) (authinterfaces.IUserProvider, bool) {
	p, ok := auth.(authinterfaces.IUserProvider)
	return p, ok
}

// implementsSessionProvider checks if auth implements ISessionProvider.
func implementsSessionProvider(auth MastraAuthProvider) (authinterfaces.ISessionProvider, bool) {
	p, ok := auth.(authinterfaces.ISessionProvider)
	return p, ok
}

// implementsSSOProvider checks if auth implements ISSOProvider.
func implementsSSOProvider(auth MastraAuthProvider) (authinterfaces.ISSOProvider, bool) {
	p, ok := auth.(authinterfaces.ISSOProvider)
	return p, ok
}

// implementsCredentialsProvider checks if auth implements ICredentialsProvider.
func implementsCredentialsProvider(auth MastraAuthProvider) (authinterfaces.ICredentialsProvider, bool) {
	p, ok := auth.(authinterfaces.ICredentialsProvider)
	return p, ok
}

// implementsACLProvider checks if auth implements IACLProvider.
func implementsACLProvider(auth MastraAuthProvider) (eeinterfaces.IACLProvider[eeinterfaces.EEUser], bool) {
	p, ok := auth.(eeinterfaces.IACLProvider[eeinterfaces.EEUser])
	return p, ok
}

// MastraCloudAuthMarker is an interface that marks a provider as MastraCloudAuth
// (exempt from license requirement).
type MastraCloudAuthMarker interface {
	IsMastraCloudAuth() bool
}

// SimpleAuthMarker is an interface that marks a provider as SimpleAuth
// (exempt from license requirement, for development/testing).
type SimpleAuthMarker interface {
	IsSimpleAuth() bool
}

// isMastraCloudAuth checks if auth provider is MastraCloudAuth.
func isMastraCloudAuth(auth MastraAuthProvider) bool {
	if m, ok := auth.(MastraCloudAuthMarker); ok {
		return m.IsMastraCloudAuth()
	}
	return false
}

// isSimpleAuth checks if auth provider is SimpleAuth.
func isSimpleAuth(auth MastraAuthProvider) bool {
	if m, ok := auth.(SimpleAuthMarker); ok {
		return m.IsSimpleAuth()
	}
	return false
}

// BuildCapabilitiesOptions are options for building capabilities.
type BuildCapabilitiesOptions struct {
	// RBAC is the RBAC provider for role-based access control (EE feature).
	// Separate from the auth provider to allow mixing different providers.
	RBAC eeinterfaces.IRBACProvider[eeinterfaces.EEUser]
}

// BuildCapabilitiesResult holds either a public or authenticated capabilities response.
type BuildCapabilitiesResult struct {
	// Public is set when the user is not authenticated.
	Public *PublicAuthCapabilities
	// Authenticated is set when the user is authenticated.
	Authenticated *AuthenticatedCapabilities
}

// BuildCapabilities builds a capabilities response based on auth configuration and request state.
//
// This function determines what capabilities are available and, if the user
// is authenticated, includes their user info and access permissions.
func BuildCapabilities(
	auth MastraAuthProvider,
	r *http.Request,
	options *BuildCapabilitiesOptions,
) (*BuildCapabilitiesResult, error) {
	// No auth configured - disabled
	if auth == nil {
		return &BuildCapabilitiesResult{
			Public: &PublicAuthCapabilities{Enabled: false, Login: nil},
		}, nil
	}

	// Determine if EE features are available
	// SimpleAuth, MastraCloudAuth, and dev environments are exempt from license requirement
	hasLicense := IsLicenseValid()
	isCloud := isMastraCloudAuth(auth)
	isSimple := isSimpleAuth(auth)
	isDev := IsDevEnvironment()
	isLicensedOrCloud := hasLicense || isCloud || isSimple || isDev

	// Build login configuration (always public)
	var login *LoginConfig

	_, hasSSO := implementsSSOProvider(auth)
	hasSSO = hasSSO && isLicensedOrCloud

	_, hasCredentials := implementsCredentialsProvider(auth)
	hasCredentials = hasCredentials && isLicensedOrCloud

	// Check if sign-up is enabled (defaults to true)
	signUpEnabled := true
	if cp, ok := implementsCredentialsProvider(auth); ok {
		signUpEnabled = cp.IsSignUpEnabled()
	}

	if hasSSO && hasCredentials {
		ssoProvider, _ := implementsSSOProvider(auth)
		ssoConfig := ssoProvider.GetLoginButtonConfig()
		login = &LoginConfig{
			Type:          "both",
			SignUpEnabled: &signUpEnabled,
			SSO: &SSOConfig{
				Provider: ssoConfig.Provider,
				Text:     ssoConfig.Text,
				Icon:     ssoConfig.Icon,
				URL:      "/api/auth/sso/login",
			},
		}
	} else if hasSSO {
		ssoProvider, _ := implementsSSOProvider(auth)
		ssoConfig := ssoProvider.GetLoginButtonConfig()
		login = &LoginConfig{
			Type: "sso",
			SSO: &SSOConfig{
				Provider: ssoConfig.Provider,
				Text:     ssoConfig.Text,
				Icon:     ssoConfig.Icon,
				URL:      "/api/auth/sso/login",
			},
		}
	} else if hasCredentials {
		login = &LoginConfig{
			Type:          "credentials",
			SignUpEnabled: &signUpEnabled,
		}
	}

	// Try to get current user (requires session)
	var user *eeinterfaces.EEUser
	if userProvider, ok := implementsUserProvider(auth); ok && isLicensedOrCloud {
		baseUser, err := userProvider.GetCurrentUser(r)
		if err == nil && baseUser != nil {
			user = &eeinterfaces.EEUser{User: *baseUser}
		}
		// Session invalid or expired — user stays nil
	}

	// If no user, return public response only
	if user == nil {
		return &BuildCapabilitiesResult{
			Public: &PublicAuthCapabilities{Enabled: true, Login: login},
		}, nil
	}

	// Get RBAC provider from options (if configured)
	var rbacProvider eeinterfaces.IRBACProvider[eeinterfaces.EEUser]
	if options != nil {
		rbacProvider = options.RBAC
	}
	hasRBAC := rbacProvider != nil && isLicensedOrCloud

	// Build capability flags
	_, hasUserProvider := implementsUserProvider(auth)
	_, hasSessionProvider := implementsSessionProvider(auth)
	_, hasSSOProvider := implementsSSOProvider(auth)
	_, hasACLProvider := implementsACLProvider(auth)

	capabilities := CapabilityFlags{
		User:    hasUserProvider && isLicensedOrCloud,
		Session: hasSessionProvider && isLicensedOrCloud,
		SSO:     hasSSOProvider && isLicensedOrCloud,
		RBAC:    hasRBAC,
		ACL:     hasACLProvider && isLicensedOrCloud,
	}

	// Get roles/permissions from RBAC provider (if available)
	var access *UserAccess
	if hasRBAC && rbacProvider != nil {
		roles, err := rbacProvider.GetRoles(*user)
		if err == nil {
			permissions, err := rbacProvider.GetPermissions(*user)
			if err == nil {
				access = &UserAccess{
					Roles:       roles,
					Permissions: permissions,
				}
			}
		}
		// RBAC failed, continue without access info
	}

	return &BuildCapabilitiesResult{
		Authenticated: &AuthenticatedCapabilities{
			PublicAuthCapabilities: PublicAuthCapabilities{
				Enabled: true,
				Login:   login,
			},
			User: AuthenticatedUser{
				ID:        user.ID,
				Email:     user.Email,
				Name:      user.Name,
				AvatarURL: user.AvatarURL,
			},
			Capabilities: capabilities,
			Access:        access,
		},
	}, nil
}
