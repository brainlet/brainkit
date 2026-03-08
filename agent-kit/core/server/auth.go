// Ported from: packages/core/src/server/auth.ts
package server

import (
	"net/http"

	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// MastraAuthProviderOptions holds configuration for creating a MastraAuthProvider.
type MastraAuthProviderOptions struct {
	// Name is the display name for this auth provider.
	Name string
	// AuthorizeUser is an optional user authorization callback.
	AuthorizeUser func(user any, r *http.Request) (bool, error)
	// Protected paths for the auth provider.
	Protected []PathRule
	// Public paths for the auth provider.
	Public []PathRule
}

// MastraAuthProvider is the abstract interface for authentication providers.
// In TypeScript this was an abstract class extending MastraBase.
// In Go, this is an interface. Concrete providers embed MastraAuthProviderBase.
type MastraAuthProvider interface {
	// AuthenticateToken authenticates a token and returns the user payload.
	// Returns nil if authentication fails.
	AuthenticateToken(token string, r *http.Request) (any, error)

	// AuthorizeUser checks if a user is authorized for the given request.
	AuthorizeUser(user any, r *http.Request) (bool, error)

	// GetProtected returns the protected path rules.
	GetProtected() []PathRule

	// GetPublic returns the public path rules.
	GetPublic() []PathRule

	// Logger returns the underlying logger.
	Logger() logger.IMastraLogger
}

// MastraAuthProviderBase provides the shared state and default implementations
// for MastraAuthProvider. Concrete providers embed this struct.
// This is the Go equivalent of the abstract class MastraAuthProvider<TUser> in TypeScript.
type MastraAuthProviderBase struct {
	*agentkit.MastraBase

	// Protected paths for the auth provider.
	ProtectedPaths []PathRule
	// Public paths for the auth provider.
	PublicPaths []PathRule

	// authorizeUserFn is the optional user authorization callback set via options.
	authorizeUserFn func(user any, r *http.Request) (bool, error)
}

// NewMastraAuthProviderBase creates a new MastraAuthProviderBase with the given options.
func NewMastraAuthProviderBase(opts *MastraAuthProviderOptions) *MastraAuthProviderBase {
	name := ""
	if opts != nil {
		name = opts.Name
	}

	base := &MastraAuthProviderBase{
		MastraBase: agentkit.NewMastraBase(agentkit.MastraBaseOptions{
			Component: logger.RegisteredLoggerAuth,
			Name:      name,
		}),
	}

	if opts != nil {
		base.RegisterOptions(opts)
	}

	return base
}

// RegisterOptions applies the given options to the provider base.
// This mirrors the protected registerOptions method in TypeScript.
func (b *MastraAuthProviderBase) RegisterOptions(opts *MastraAuthProviderOptions) {
	if opts == nil {
		return
	}

	if opts.AuthorizeUser != nil {
		b.authorizeUserFn = opts.AuthorizeUser
	}
	if opts.Protected != nil {
		b.ProtectedPaths = opts.Protected
	}
	if opts.Public != nil {
		b.PublicPaths = opts.Public
	}
}

// GetProtected returns the protected path rules.
func (b *MastraAuthProviderBase) GetProtected() []PathRule {
	return b.ProtectedPaths
}

// GetPublic returns the public path rules.
func (b *MastraAuthProviderBase) GetPublic() []PathRule {
	return b.PublicPaths
}

// AuthorizeUser is the default authorization implementation.
// If an authorizeUserFn was provided via options, it delegates to that.
// Otherwise, concrete providers must override this by implementing MastraAuthProvider.
func (b *MastraAuthProviderBase) AuthorizeUser(user any, r *http.Request) (bool, error) {
	if b.authorizeUserFn != nil {
		return b.authorizeUserFn(user, r)
	}
	// Default: deny. Concrete providers should override.
	return false, nil
}
