// Ported from: packages/core/src/server/base.ts
package server

import (
	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// MastraServerBase is the base type for server adapters that provides app storage
// and retrieval.
//
// This type embeds MastraBase to get logging capabilities and provides a
// framework-agnostic way to store and retrieve the server app instance
// (e.g., a chi router, an http.ServeMux, etc.).
//
// Server adapters extend this base by embedding it while adding their
// framework-specific route registration and middleware handling.
//
// In TypeScript this was: abstract class MastraServerBase<TApp> extends MastraBase
// In Go, the TApp generic is represented as any. Callers use GetApp with a type assertion.
type MastraServerBase struct {
	*agentkit.MastraBase
	app any
}

// MastraServerBaseOptions holds the constructor options for MastraServerBase.
type MastraServerBaseOptions struct {
	// App is the server framework app instance (e.g., *chi.Mux, *http.ServeMux).
	App any
	// Name is the display name for this server. Default: "Server".
	Name string
}

// NewMastraServerBase creates a new MastraServerBase with the given options.
func NewMastraServerBase(opts MastraServerBaseOptions) *MastraServerBase {
	name := opts.Name
	if name == "" {
		name = "Server"
	}

	s := &MastraServerBase{
		MastraBase: agentkit.NewMastraBase(agentkit.MastraBaseOptions{
			Component: logger.RegisteredLoggerServer,
			Name:      name,
		}),
		app: opts.App,
	}

	s.Logger().Debug("Server app set")

	return s
}

// GetApp returns the underlying server app instance.
// Callers are responsible for type-asserting the result to the correct type.
//
// Example:
//
//	mux, ok := server.GetApp().(*chi.Mux)
func (s *MastraServerBase) GetApp() any {
	return s.app
}

// App returns the underlying server app instance for subclasses/embedders.
// This mirrors the protected getter in the TypeScript original.
func (s *MastraServerBase) App() any {
	return s.app
}
