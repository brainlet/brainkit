package packages

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	runtimecap "github.com/brainlet/brainkit/modulecap/runtime"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
)

type recordingDeployer struct {
	source string
	code   string
	cfg    types.DeployConfig
}

func (d *recordingDeployer) DeployArtifact(_ context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error) {
	var cfg types.DeployConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	types.WithNormalizedJS()(&cfg)
	d.source = source
	d.code = code
	d.cfg = cfg
	return nil, nil
}

func (d *recordingDeployer) Teardown(context.Context, string) (int, error) { return 0, nil }

func (d *recordingDeployer) ListDeployments() []runtimecap.DeploymentInfo { return nil }

func TestPackageDeployMarksBundledCodeAsNormalizedJS(t *testing.T) {
	deployer := &recordingDeployer{}
	domain := NewDomain(deployer, nil, nil)

	manifest, err := json.Marshal(map[string]string{
		"name":    "pkg",
		"version": "1.0.0",
		"entry":   "index.ts",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = domain.Deploy(ctx, packagemsg.PackageDeployMsg{
		Manifest: manifest,
		Files: map[string]string{
			"index.ts": `bus.on("ping", function(msg) { msg.reply({ ok: true }); });`,
		},
	})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if deployer.cfg.EffectiveArtifactKind() != types.DeployArtifactNormalizedJS {
		t.Fatalf("package deploy must pass normalized JS artifacts to the runtime: %#v", deployer.cfg)
	}
	if deployer.cfg.PackageName != "pkg" {
		t.Fatalf("package name = %q, want pkg", deployer.cfg.PackageName)
	}
	if deployer.source != "pkg.ts" {
		t.Fatalf("source = %q, want pkg.ts", deployer.source)
	}
	if deployer.code == "" {
		t.Fatalf("bundled code was empty")
	}
}
