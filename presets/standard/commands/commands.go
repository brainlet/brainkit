// Package commands builds the light standard command/control-plane module set.
//
// It intentionally avoids the JS runtime and source package deployment. Import
// presets/standard/packages or use standard.FullCommandSet when package.deploy
// and runtime TypeScript support are required.
package commands

import (
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/presets/standard/core"
)

// Set returns fresh module instances for the light command/control-plane
// profile.
func Set() []bkmodule.Module {
	return core.Set()
}
