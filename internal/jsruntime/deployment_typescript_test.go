package jsruntime

import (
	"strings"
	"testing"

	"github.com/brainlet/brainkit/internal/embed/typescript"
	"github.com/brainlet/brainkit/internal/types"
)

func TestPrepareDeployCodeTranspilesTSSource(t *testing.T) {
	m := &DeploymentManager{sourcePreparer: func(source, code string) (string, error) {
		return typescript.TranspileTS(code, source)
	}}
	code := `import { output } from "kit";

interface Config {
  value: string;
}

type Result = { value: string };

const cfg: Config = { value: "ok" };
const result: Result = { value: cfg.value };
output(result);`

	got, err := m.prepareDeployCode("raw.ts", code, types.DeployConfig{})
	if err != nil {
		t.Fatalf("prepareDeployCode: %v", err)
	}
	for _, forbidden := range []string{"interface Config", "type Result", ": Config", ": Result", `import { output } from "kit"`} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("prepared code still contains %q:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, `value: "ok"`) || !strings.Contains(got, "output(result)") {
		t.Fatalf("prepared code lost runtime statements:\n%s", got)
	}
}

func TestPrepareDeployCodeLeavesNormalizedJSAlone(t *testing.T) {
	m := &DeploymentManager{}
	code := `const alreadyBundled = true;`

	got, err := m.prepareDeployCode("bundle.ts", code, types.DeployConfig{ArtifactKind: types.DeployArtifactNormalizedJS})
	if err != nil {
		t.Fatalf("prepareDeployCode: %v", err)
	}
	if got != code {
		t.Fatalf("normalized JS was modified:\n%s", got)
	}
}

func TestPrepareDeployCodeRejectsRawTSWithoutSourcePreparer(t *testing.T) {
	m := &DeploymentManager{sourcePreparer: nil}

	_, err := m.prepareDeployCode("raw.ts", `interface Config { value: string }`, types.DeployConfig{})
	if err == nil {
		t.Fatal("prepareDeployCode error = nil, want disabled TypeScript source error")
	}
	if !strings.Contains(err.Error(), "typescript source deployment is disabled") {
		t.Fatalf("prepareDeployCode error = %v, want disabled TypeScript source message", err)
	}
}
