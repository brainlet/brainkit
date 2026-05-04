// Package core registers the light standard YAML module catalog.
//
// This profile contains command/control-plane modules that do not link the
// embedded JS runtime, TypeScript bundler, HTTP gateway, SQL-backed standard
// stores, MCP, plugins, or dev harnesses.
package core

import (
	_ "github.com/brainlet/brainkit/modules/agents"
	_ "github.com/brainlet/brainkit/modules/control"
	_ "github.com/brainlet/brainkit/modules/health"
	_ "github.com/brainlet/brainkit/modules/messaging"
	_ "github.com/brainlet/brainkit/modules/metrics"
	_ "github.com/brainlet/brainkit/modules/reference"
	_ "github.com/brainlet/brainkit/modules/registry"
	_ "github.com/brainlet/brainkit/modules/secrets"
	_ "github.com/brainlet/brainkit/modules/tools"
)
