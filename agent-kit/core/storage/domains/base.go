// Ported from: packages/core/src/storage/domains/base.ts
package domains

import (
	"context"

	agentkit "github.com/brainlet/brainkit/agent-kit/core"
)

// StorageDomain is the base interface for all storage domains.
// It provides a common interface for initialization and data clearing.
//
// In TypeScript this is an abstract class extending MastraBase. In Go we model
// it as an interface — concrete adapters embed *agentkit.MastraBase and
// implement these methods.
type StorageDomain interface {
	// Init initializes the storage domain.
	// This should create any necessary tables/collections.
	// Default behavior is a no-op — adapters override if they need initialization.
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data from this storage domain.
	// This is a destructive operation — use with caution.
	// Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error
}

// StorageDomainBase provides the default no-op Init implementation that
// concrete storage domains can embed, mirroring the TypeScript base class
// behavior where init() is a default no-op.
type StorageDomainBase struct {
	*agentkit.MastraBase
}

// Init is a default no-op — adapters override if they need to create tables/collections.
func (s *StorageDomainBase) Init(_ context.Context) error {
	return nil
}
