// Package commands registers the standard command/runtime YAML module catalog.
//
// It is the YAML-registration counterpart to presets/standard.CommandSet:
// core command modules plus source package deployment.
package commands

import (
	_ "github.com/brainlet/brainkit/server/standard/core"
	_ "github.com/brainlet/brainkit/server/standard/packages"
)
