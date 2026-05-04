// Package full registers every built-in standard YAML module profile.
//
// Use this for product CLIs and batteries-included binaries. Smaller server
// binaries should prefer the specific profiles under server/standard.
package full

import (
	_ "github.com/brainlet/brainkit/server/standard/automation"
	_ "github.com/brainlet/brainkit/server/standard/commands"
	_ "github.com/brainlet/brainkit/server/standard/dev"
	_ "github.com/brainlet/brainkit/server/standard/integrations"
	_ "github.com/brainlet/brainkit/server/standard/observability"
	_ "github.com/brainlet/brainkit/server/standard/server"
)
