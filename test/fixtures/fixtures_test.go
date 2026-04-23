package fixtures_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/fixtures"
)

func TestMain(m *testing.M) {
	if _, err := testutil.ResolvePodmanSocket(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: cannot resolve podman socket for fixtures: %v\n", err)
		fmt.Fprintf(os.Stderr, "       run `make podman-ensure` to start the brainkit podman machine.\n")
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestFixtures(t *testing.T) {
	runner := fixtures.NewRunner(fixtures.FixturesRoot(t))
	runner.RunAll(t)
}
