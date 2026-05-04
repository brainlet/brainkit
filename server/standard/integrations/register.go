// Package integrations registers integration-oriented YAML module names.
//
// These modules are useful in product/server binaries but are not part of the
// light command/runtime catalog.
package integrations

import (
	_ "github.com/brainlet/brainkit/modules/discovery"
	_ "github.com/brainlet/brainkit/modules/mcp"
	_ "github.com/brainlet/brainkit/modules/plugins"
	_ "github.com/brainlet/brainkit/modules/topology"
)
