// Package server registers server-facing YAML module names.
//
// This profile covers modules commonly needed by an HTTP-facing server binary:
// the gateway and provider/storage/vector probes. It does not register the
// command/runtime catalog; import server/standard/commands for that.
package server

import (
	_ "github.com/brainlet/brainkit/modules/gateway"
	_ "github.com/brainlet/brainkit/modules/probes"
)
