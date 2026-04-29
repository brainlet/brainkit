// Package standard composes the built-in command modules that make up the
// default Kit bus surface.
package standard

import (
	bkmodule "github.com/brainlet/brainkit/module"
	agentsmod "github.com/brainlet/brainkit/modules/agents"
	controlmod "github.com/brainlet/brainkit/modules/control"
	evalmod "github.com/brainlet/brainkit/modules/eval"
	healthmod "github.com/brainlet/brainkit/modules/health"
	jsruntimemod "github.com/brainlet/brainkit/modules/jsruntime"
	messagingmod "github.com/brainlet/brainkit/modules/messaging"
	metricsmod "github.com/brainlet/brainkit/modules/metrics"
	packagesmod "github.com/brainlet/brainkit/modules/packages"
	referencemod "github.com/brainlet/brainkit/modules/reference"
	registrymod "github.com/brainlet/brainkit/modules/registry"
	secretsmod "github.com/brainlet/brainkit/modules/secrets"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
)

// CommandSet returns the lightweight standard bus command modules. Each call
// returns fresh module instances so callers can append, reorder, or mount the
// result without sharing mutable module state.
func CommandSet() []bkmodule.Module {
	return []bkmodule.Module{
		jsruntimemod.New(),
		agentsmod.New(),
		referencemod.New(),
		controlmod.New(),
		evalmod.New(),
		healthmod.New(),
		messagingmod.New(),
		metricsmod.New(),
		registrymod.New(),
		secretsmod.New(),
		toolsmod.New(),
		packagesmod.New(),
	}
}
