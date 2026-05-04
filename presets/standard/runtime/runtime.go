// Package runtime builds the standard JS runtime/eval module set.
//
// This profile intentionally links the embedded JS runtime but not package
// source builders. Import presets/standard/packages for source package deploy.
package runtime

import (
	bkmodule "github.com/brainlet/brainkit/module"
	evalmod "github.com/brainlet/brainkit/modules/eval"
	jsruntimemod "github.com/brainlet/brainkit/modules/jsruntime"
)

// Set returns fresh module instances for the JS runtime/eval profile.
func Set() []bkmodule.Module {
	return []bkmodule.Module{
		jsruntimemod.New(),
		evalmod.New(),
	}
}
