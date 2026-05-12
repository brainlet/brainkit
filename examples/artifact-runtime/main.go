// Command artifact-runtime demonstrates the normalized-JS-only runtime profile.
//
// It mounts presets/standard.ArtifactRuntimeSet(), then mounts a tiny module
// that consumes only brainkit.core.artifact_deployer. The module deploys a
// pre-normalized JavaScript artifact directly into the runtime, without the
// packages module, esbuild, or the runtime TypeScript source preparer.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/brainlet/brainkit"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modulecap/runtime"
	"github.com/brainlet/brainkit/modules/eval/evalmsg"
	"github.com/brainlet/brainkit/presets/standard"
	"github.com/brainlet/brainkit/sdk"
)

const artifactSource = "artifact-runtime-demo.ts"

const normalizedArtifact = `
bus.on("artifact.echo", (msg) => {
	const name = msg.payload && msg.payload.name ? msg.payload.name : "world";
	msg.reply({
		greeting: "hello, " + name,
		profile: "artifact-runtime",
	});
});
`

const rawTypeScript = `
interface Config {
	value: string;
}
const cfg: Config = { value: "raw-typescript" };
output(cfg);
`

type artifactDemoModule struct{}

func (artifactDemoModule) ID() string { return "artifact-runtime-demo" }

func (artifactDemoModule) Mount(ctx context.Context, host bkmodule.Host) error {
	deployer, err := bkmodule.RequireCapability[runtimecap.ArtifactDeployer](host, bkmodule.CapabilityArtifactDeployer)
	if err != nil {
		return fmt.Errorf("artifact-runtime-demo: %w", err)
	}
	if _, err := deployer.DeployArtifact(ctx, artifactSource, normalizedArtifact); err != nil {
		return fmt.Errorf("artifact-runtime-demo: deploy normalized artifact: %w", err)
	}
	host.Scope().Defer(func(ctx context.Context) error {
		_, err := deployer.Teardown(ctx, artifactSource)
		return err
	})

	if _, err := deployer.DeployArtifact(ctx, "raw-typescript.ts", rawTypeScript); err == nil {
		return fmt.Errorf("artifact-runtime-demo: raw TypeScript was accepted by the artifact-only deploy path")
	}
	return nil
}

func (artifactDemoModule) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "artifact-runtime-demo",
		Status:  bkmodule.StatusBeta,
		Summary: "Deploys a normalized JavaScript artifact through the artifact-only runtime capability.",
		Requires: []string{
			"jsruntime",
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[runtimecap.ArtifactDeployer](bkmodule.CapabilityArtifactDeployer),
		},
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("artifact-runtime: %v", err)
	}
}

func run() error {
	modules := append(standard.ArtifactRuntimeSet(), artifactDemoModule{})
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "artifact-runtime",
		Transport: brainkit.Memory(),
		FSRoot:    ".",
		Modules:   modules,
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.artifact-runtime-demo.artifact.echo",
		Payload: json.RawMessage(`{"name":"world"}`),
	}, brainkit.WithCallTimeout(2*time.Second))
	if err != nil {
		return fmt.Errorf("call artifact handler: %w", err)
	}

	evalResp, err := evalmsg.CallKitEval(kit, ctx, evalmsg.KitEvalMsg{
		Mode:   "js",
		Source: "artifact-runtime-eval.js",
		Code:   `return "eval-js-ok";`,
	}, sdk.WithCallTimeout(2*time.Second))
	if err != nil {
		return fmt.Errorf("eval js: %w", err)
	}

	fmt.Println(string(reply))
	fmt.Println(evalResp.Result)
	fmt.Println("raw TypeScript rejected by artifact-only deploy path")
	return nil
}
