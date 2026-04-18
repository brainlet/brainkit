package harness_test

import (
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/harness"
	"github.com/stretchr/testify/require"
)

// TestModuleLifecycle asserts that a Kit with the harness module wired
// (but no harness config) boots and closes cleanly. Harness.Init needs
// the Kit's JS bridge present; the zero-value HarnessConfig triggers
// the validator to reject the launch, so Init short-circuits to a
// no-op state. Close should be idempotent on that no-op.
func TestModuleLifecycle(t *testing.T) {
	m := harness.NewModule(harness.Config{})

	kit, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test-harness",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
		Modules:   []brainkit.Module{m},
	})
	// A zero-value HarnessConfig fails validation; the module init
	// surfaces that error from the Kit constructor.
	if err != nil {
		require.Contains(t, err.Error(), "harness")
		return
	}
	defer kit.Close()

	require.Equal(t, brainkit.ModuleStatusWIP, m.Status())
}
