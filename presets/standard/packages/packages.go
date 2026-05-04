// Package packages builds the standard source package deployment profile.
//
// It composes the JS runtime/eval profile with the packages command module and
// the standard esbuild package builder. Import presets/standard/runtime when a
// binary needs the JS runtime without source-package deployment.
package packages

import (
	bkmodule "github.com/brainlet/brainkit/module"
	packagesmod "github.com/brainlet/brainkit/modules/packages"
	_ "github.com/brainlet/brainkit/modules/packages/bundlers/esbuild"
	runtimepreset "github.com/brainlet/brainkit/presets/standard/runtime"
)

// Set returns fresh module instances for JS runtime/eval plus source package
// deployment.
func Set() []bkmodule.Module {
	mods := runtimepreset.Set()
	mods = append(mods, packagesmod.New())
	return mods
}
