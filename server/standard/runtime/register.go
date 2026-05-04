// Package runtime registers JS/runtime YAML module names.
//
// This profile owns the embedded JS runtime and eval surface. It intentionally
// links QuickJS/Wazero through modules/jsruntime, but it does not register the
// packages module or esbuild source builder. Import server/standard/packages
// when YAML should accept source package deployment too.
package runtime

import (
	_ "github.com/brainlet/brainkit/modules/eval"
	_ "github.com/brainlet/brainkit/modules/jsruntime"
)
