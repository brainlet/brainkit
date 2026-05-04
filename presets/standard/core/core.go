// Package core builds the light standard command/control-plane module set.
//
// It intentionally avoids the embedded JS runtime, package bundler, gateway,
// SQL-backed standard stores, MCP, plugins, and dev harnesses.
package core

import (
	bkmodule "github.com/brainlet/brainkit/module"
	agentsmod "github.com/brainlet/brainkit/modules/agents"
	controlmod "github.com/brainlet/brainkit/modules/control"
	healthmod "github.com/brainlet/brainkit/modules/health"
	messagingmod "github.com/brainlet/brainkit/modules/messaging"
	metricsmod "github.com/brainlet/brainkit/modules/metrics"
	referencemod "github.com/brainlet/brainkit/modules/reference"
	registrymod "github.com/brainlet/brainkit/modules/registry"
	secretsmod "github.com/brainlet/brainkit/modules/secrets"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
)

// Set returns fresh module instances for the light command/control-plane
// profile.
func Set() []bkmodule.Module {
	return []bkmodule.Module{
		agentsmod.New(),
		referencemod.New(),
		controlmod.New(),
		healthmod.New(),
		messagingmod.New(),
		metricsmod.New(),
		registrymod.New(),
		secretsmod.New(),
		toolsmod.New(),
	}
}
