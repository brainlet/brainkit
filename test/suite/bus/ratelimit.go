package bus

import (
	"testing"

	"github.com/brainlet/brainkit/test/suite"
)

// testBusRateLimitExceeds — RBAC rate limits have been removed.
func testBusRateLimitExceeds(t *testing.T, _ *suite.TestEnv) {
	t.Skip("RBAC rate limits have been removed")
}
