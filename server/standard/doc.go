// Package standard documents Brainkit's standard server module profiles.
//
// This package intentionally has no side-effect registrations. Import explicit
// profiles in binaries that want configfile.Load to accept built-in modules in
// YAML:
//
//	import _ "github.com/brainlet/brainkit/server/standard/full"
//
// Custom binaries should usually prefer narrower profiles:
//
//	import _ "github.com/brainlet/brainkit/server/standard/commands"
//	import _ "github.com/brainlet/brainkit/server/standard/packages"
//	import _ "github.com/brainlet/brainkit/server/standard/server"
//
// Other profiles include observability, automation, integrations, dev, core,
// runtime, packages, and full.
package standard
