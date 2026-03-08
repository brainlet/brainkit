// Ported from: packages/core/src/di/index.ts
package di

// The TypeScript di/index.ts is a pure re-export file:
//
//   export { RequestContext, MASTRA_RESOURCE_ID_KEY, MASTRA_THREAD_ID_KEY } from '../request-context';
//
// In Go, re-exports don't exist as a language construct. Consumers should
// import the requestcontext package directly:
//
//   import "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
//
// The following aliases are provided for convenience and to maintain a 1:1
// mapping with the TypeScript module structure.

import (
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// RequestContext is an alias for requestcontext.RequestContext.
// See the requestcontext package for full documentation.
type RequestContext = requestcontext.RequestContext

// Key constants re-exported from the requestcontext package.
var (
	// MastraResourceIDKey is the context key for the resource ID.
	MastraResourceIDKey = requestcontext.MastraResourceIDKey

	// MastraThreadIDKey is the context key for the thread ID.
	MastraThreadIDKey = requestcontext.MastraThreadIDKey
)

// NewRequestContext creates a new RequestContext.
// Delegates to requestcontext.NewRequestContext.
func NewRequestContext() *RequestContext {
	return requestcontext.NewRequestContext()
}
