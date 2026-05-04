package deploy

import (
	"encoding/json"
	"strings"

	"github.com/brainlet/brainkit/modules/packages/packagemsg"
)

// pkgDeploy builds a PackageDeployMsg for the inline (single-file) deploy path.
// Name is derived by stripping ".ts" from entry.
func pkgDeploy(entry, code string) packagemsg.PackageDeployMsg {
	name := strings.TrimSuffix(entry, ".ts")
	manifest, _ := json.Marshal(map[string]string{"name": name, "entry": entry})
	return packagemsg.PackageDeployMsg{
		Manifest: manifest,
		Files:    map[string]string{entry: code},
	}
}

// pkgTeardown builds a PackageTeardownMsg from a source filename.
func pkgTeardown(source string) packagemsg.PackageTeardownMsg {
	return packagemsg.PackageTeardownMsg{Name: strings.TrimSuffix(source, ".ts")}
}

func tsServiceTopic(source, topic string) string {
	name := strings.TrimSuffix(source, ".ts")
	name = strings.ReplaceAll(name, "/", ".")
	return "ts." + name + "." + topic
}
