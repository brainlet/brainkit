// Package commands registers the light standard command/control-plane YAML
// module catalog.
//
// It is the YAML-registration counterpart to presets/standard.CommandSet:
// core command modules only. Import server/standard/packages when
// package.deploy and runtime TypeScript support are required.
package commands

import (
	_ "github.com/brainlet/brainkit/server/standard/core"
)
