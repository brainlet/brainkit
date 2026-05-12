// Package artifactruntime builds the standard artifact-only JS runtime/eval set.
//
// This profile links the embedded JS runtime and eval surface without the
// runtime TypeScript source preparer. It accepts normalized JavaScript artifacts
// and rejects raw .ts source deploys.
package artifactruntime

import (
	bkmodule "github.com/brainlet/brainkit/module"
	evalmod "github.com/brainlet/brainkit/modules/eval"
	jsruntimeartifact "github.com/brainlet/brainkit/modules/jsruntime/artifact"
)

// Set returns fresh module instances for the artifact-only JS runtime/eval
// profile.
func Set() []bkmodule.Module {
	return []bkmodule.Module{
		jsruntimeartifact.New(),
		evalmod.New(),
	}
}
