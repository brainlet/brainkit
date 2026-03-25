package fixtures_test

import (
	"testing"
)

// TestPluginFixtures runs .ts fixtures that require a Node with a plugin
// subprocess connected via NATS. Fixtures live in fixtures/ts/plugin/.
//
// The test plugin registers tools and handles events. The .ts fixture
// calls those tools and verifies the responses.
//
// Requires: Podman (NATS container), test plugin binary
func TestPluginFixtures(t *testing.T) {
	// TODO Phase 2: build plugin, start NATS, create Node, walk fixtures/ts/plugin/
	t.Skip("plugin fixtures not yet implemented (Phase 2)")
}
