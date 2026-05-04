// Package standard registers Brainkit's full standard server module catalog.
//
// Import it for side effects in binaries that want configfile.Load to accept the
// built-in modules in YAML:
//
//	import _ "github.com/brainlet/brainkit/server/standard"
//
// Custom binaries should prefer the narrower profiles:
//
//	import _ "github.com/brainlet/brainkit/server/standard/commands"
//	import _ "github.com/brainlet/brainkit/server/standard/server"
//
// Other profiles include observability, automation, integrations, dev, core,
// runtime, packages, and full.
package standard
