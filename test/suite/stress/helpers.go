package stress

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/packages"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	secretsmod "github.com/brainlet/brainkit/modules/secrets"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/sdk"
)

func stressModules() []brainkit.Module {
	return []brainkit.Module{secretsmod.New(), toolsmod.New(), packages.New()}
}

func stressTeardown(t *testing.T, rt sdk.CallerRuntime, sourceOrName string) error {
	t.Helper()
	name := strings.TrimSuffix(sourceOrName, ".ts")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := sdk.Call[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](rt, ctx, packagemsg.PackageTeardownMsg{Name: name})
	if err != nil {
		t.Logf("stress teardown %s: %v", name, err)
	}
	return err
}

func cleanupStressDeployments(t *testing.T, rt sdk.CallerRuntime) {
	t.Helper()
	for _, dep := range testutil.ListDeployments(t, rt) {
		name := dep.Name
		if name == "" {
			name = dep.Source
		}
		_ = stressTeardown(t, rt, name)
	}
}
