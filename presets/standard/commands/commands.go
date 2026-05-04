// Package commands builds the standard command/runtime module set.
//
// It composes the light core command plane with source package deployment
// modules.
package commands

import (
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/presets/standard/core"
	packagepreset "github.com/brainlet/brainkit/presets/standard/packages"
)

// Set returns fresh module instances for the standard command/runtime profile.
func Set() []bkmodule.Module {
	mods := core.Set()
	mods = append(mods, packagepreset.Set()...)
	return mods
}
