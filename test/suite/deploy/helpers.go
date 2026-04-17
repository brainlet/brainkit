package deploy

import (
	"encoding/json"
	"strings"

	"github.com/brainlet/brainkit/sdk"
)

// pkgDeploy builds a PackageDeployMsg for the inline (single-file) deploy path.
// Name is derived by stripping ".ts" from entry.
func pkgDeploy(entry, code string) sdk.PackageDeployMsg {
	name := strings.TrimSuffix(entry, ".ts")
	manifest, _ := json.Marshal(map[string]string{"name": name, "entry": entry})
	return sdk.PackageDeployMsg{
		Manifest: manifest,
		Files:    map[string]string{entry: code},
	}
}

// pkgTeardown builds a PackageTeardownMsg from a source filename.
func pkgTeardown(source string) sdk.PackageTeardownMsg {
	return sdk.PackageTeardownMsg{Name: strings.TrimSuffix(source, ".ts")}
}
