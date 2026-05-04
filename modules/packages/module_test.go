package packages

import (
	"context"
	"encoding/json"
	"strings"
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

type recordingBuilder struct {
	req BuildRequest
}

func (b *recordingBuilder) BuildPackage(_ context.Context, req BuildRequest, _ PluginChecker, _ SecretChecker) (BuiltPackage, error) {
	b.req = req
	return BuiltPackage{
		Name:    "pkg",
		Version: "1.0.0",
		Source:  "pkg.ts",
		Code:    `bus.on("ping", function(msg) { msg.reply({ ok: true }); });`,
	}, nil
}

func TestPackageDeployMarksBundledCodeAsNormalizedJS(t *testing.T) {
	deployer := &recordingDeployer{}
	builder := &recordingBuilder{}
	domain := NewDomain(deployer, nil, nil, withDomainPackageBuilder(builder))

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
	if string(builder.req.Manifest) == "" || len(builder.req.Files) == 0 {
		t.Fatalf("builder did not receive package deploy payload: %#v", builder.req)
	}
}

func TestPackageDeployRequiresPackageBuilder(t *testing.T) {
	deployer := &recordingDeployer{}
	domain := NewDomain(deployer, nil, nil, withDomainPackageBuilder(nil))

	manifest, err := json.Marshal(map[string]string{
		"name":  "pkg",
		"entry": "index.ts",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = domain.Deploy(context.Background(), packagemsg.PackageDeployMsg{
		Manifest: manifest,
		Files:    map[string]string{"index.ts": `output("x");`},
	})
	if err == nil {
		t.Fatal("expected missing package builder error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "modules/packages/bundlers/esbuild") {
		t.Fatalf("error = %q, want esbuild bundler import hint", got)
	}
}
