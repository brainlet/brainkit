// Package artifactruntime registers artifact-only JS runtime YAML module names.
//
// This profile registers `jsruntime_artifact` and `eval`. YAML configs that
// want the artifact-only runtime should list `jsruntime_artifact: {}` before
// JS-dependent modules. The mounted module still has runtime ID `jsruntime`.
package artifactruntime

import (
	_ "github.com/brainlet/brainkit/modules/eval"
	_ "github.com/brainlet/brainkit/modules/jsruntime/artifact"
)
