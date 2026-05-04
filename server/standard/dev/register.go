// Package dev registers development/testing YAML module names.
//
// It is intentionally separate from the normal server profile because testing
// links the TypeScript bundler and harness is an experimental orchestration
// surface.
package dev

import (
	_ "github.com/brainlet/brainkit/modules/harness"
	_ "github.com/brainlet/brainkit/modules/testing"
)
