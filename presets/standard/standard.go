// Package standard composes built-in module presets for common Kit assemblies.
package standard

import (
	bkmodule "github.com/brainlet/brainkit/module"
	artifactruntimepreset "github.com/brainlet/brainkit/presets/standard/artifactruntime"
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

// ArtifactRuntimeSet returns the artifact-only JS runtime/eval modules. Raw
// `.ts` source deploys are rejected; package/tooling callers should pass
// normalized JavaScript artifacts.
func ArtifactRuntimeSet() []bkmodule.Module {
	return artifactruntimepreset.Set()
}

// PackageSet returns the JS runtime/eval/package deployment modules, including
// the standard source package builder.
func PackageSet() []bkmodule.Module {
	return packagepreset.Set()
}

// CommandSet returns the light standard bus command/control-plane modules. It
// does not link the embedded JS runtime or TypeScript package bundler. Each
// call returns fresh module instances so callers can append, reorder, or mount
// the result without sharing mutable module state.
func CommandSet() []bkmodule.Module {
	return commands.Set()
}

// FullCommandSet returns the batteries-included command/runtime modules: the
// light command/control plane plus JS runtime/eval/package deployment and the
// standard source package builder.
func FullCommandSet() []bkmodule.Module {
	mods := core.Set()
	mods = append(mods, packagepreset.Set()...)
	return mods
}
