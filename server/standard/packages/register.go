// Package packages registers source package deployment YAML module names.
//
// It composes server/standard/runtime with the packages module and the
// standard esbuild package builder.
package packages

import (
	_ "github.com/brainlet/brainkit/modules/packages"
	_ "github.com/brainlet/brainkit/modules/packages/bundlers/esbuild"
	_ "github.com/brainlet/brainkit/server/standard/runtime"
)
