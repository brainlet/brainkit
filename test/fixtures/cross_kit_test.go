package fixtures_test

import (
	"testing"
)

// TestCrossKitFixtures runs .ts fixtures that require two Kernels on a shared
// NATS transport. Fixtures live in fixtures/ts/cross-kit/.
//
// Each fixture is deployed on Kit A. Kit B has Go tools registered.
// The fixture publishes to Kit B's namespace and validates the response.
//
// Requires: Podman (NATS container)
func TestCrossKitFixtures(t *testing.T) {
	// TODO Phase 2: start NATS container, create two Kernels, walk fixtures/ts/cross-kit/
	t.Skip("cross-kit fixtures not yet implemented (Phase 2)")
}
