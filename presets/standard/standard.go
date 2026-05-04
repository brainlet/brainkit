// Package standard composes the built-in command modules that make up the
// default Kit bus surface.
package standard

import (
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/presets/standard/commands"
	"github.com/brainlet/brainkit/presets/standard/core"
	packagepreset "github.com/brainlet/brainkit/presets/standard/packages"
	runtimepreset "github.com/brainlet/brainkit/presets/standard/runtime"
)

// CoreSet returns the light standard command/control-plane modules. It does not
// link the embedded JS runtime or TypeScript package bundler.
func CoreSet() []bkmodule.Module {
	return core.Set()
}

// RuntimeSet returns the JS runtime/eval modules without source package
// builders.
func RuntimeSet() []bkmodule.Module {
	return runtimepreset.Set()
}

// PackageSet returns the JS runtime/eval/package deployment modules, including
// the standard source package builder.
func PackageSet() []bkmodule.Module {
	return packagepreset.Set()
}

// CommandSet returns the standard bus command/runtime modules. Each call
// returns fresh module instances so callers can append, reorder, or mount the
// result without sharing mutable module state.
func CommandSet() []bkmodule.Module {
	return commands.Set()
}
