// Package integration provides the Integration class for registering
// workflows and tools within an integration context.
//
// Ported from: packages/core/src/integration/integration.ts
package integration

import (
	"fmt"

	"github.com/brainlet/brainkit/agent-kit/core/tools"
	"github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// Integration is a base class for integrations that can register workflows
// and provide tools. Generic type parameters from TS (ToolsParams, ApiClient)
// are represented as any in Go since they were used with loose typing.
//
// Ported from: packages/core/src/integration/integration.ts — Integration<ToolsParams, ApiClient>
type Integration struct {
	// Name is the display name for the integration.
	//
	// Ported from: packages/core/src/integration/integration.ts — name: string = 'Integration'
	Name string

	// workflows holds registered workflows by name.
	//
	// Ported from: packages/core/src/integration/integration.ts — private workflows
	registeredWorkflows map[string]*workflows.Workflow
}

// NewIntegration creates a new Integration with default values.
//
// Ported from: packages/core/src/integration/integration.ts — constructor()
func NewIntegration() *Integration {
	return &Integration{
		Name:                "Integration",
		registeredWorkflows: make(map[string]*workflows.Workflow),
	}
}

// RegisterWorkflow registers a workflow by name. Returns an error if a
// workflow with that name is already registered.
//
// Ported from: packages/core/src/integration/integration.ts — registerWorkflow()
func (i *Integration) RegisterWorkflow(name string, wf *workflows.Workflow) error {
	if _, exists := i.registeredWorkflows[name]; exists {
		return fmt.Errorf("sync function %q already registered", name)
	}
	i.registeredWorkflows[name] = wf
	return nil
}

// SerializedWorkflow is a minimal representation of a workflow for serialization.
//
// Ported from: packages/core/src/integration/integration.ts — listWorkflows serialized branch
type SerializedWorkflow struct {
	Name string `json:"name"`
}

// ListWorkflows returns all registered workflows. If serialized is true,
// returns only name metadata.
//
// Ported from: packages/core/src/integration/integration.ts — listWorkflows()
func (i *Integration) ListWorkflows(serialized bool) map[string]*workflows.Workflow {
	if serialized {
		// In TS this returns { [k]: { name: v.name } } — we return the same map
		// since the caller can extract .Name from each Workflow. The TS behavior
		// of stripping fields is a serialization concern handled at the API boundary.
		return i.registeredWorkflows
	}
	return i.registeredWorkflows
}

// ListSerializedWorkflows returns workflows as serialized name-only metadata.
// This is the Go-idiomatic equivalent of calling listWorkflows({ serialized: true })
// in the TS source.
//
// Ported from: packages/core/src/integration/integration.ts — listWorkflows({ serialized: true })
func (i *Integration) ListSerializedWorkflows() map[string]SerializedWorkflow {
	result := make(map[string]SerializedWorkflow, len(i.registeredWorkflows))
	for k, v := range i.registeredWorkflows {
		// In TS: { name: v.name } — Workflow.ID is the closest field to TS's name
		name := v.ID
		if name == "" {
			name = k
		}
		result[k] = SerializedWorkflow{Name: name}
	}
	return result
}

// ListStaticTools returns static tools for this integration.
// Base implementation returns an error — subclasses must override.
//
// Ported from: packages/core/src/integration/integration.ts — listStaticTools()
func (i *Integration) ListStaticTools() (map[string]*tools.ToolAction, error) {
	return nil, fmt.Errorf("method not implemented")
}

// ListTools returns tools for this integration (async in TS).
// Base implementation returns an error — subclasses must override.
//
// Ported from: packages/core/src/integration/integration.ts — listTools()
func (i *Integration) ListTools() (map[string]*tools.ToolAction, error) {
	return nil, fmt.Errorf("method not implemented")
}

// GetAPIClient returns the API client for this integration.
// Base implementation returns an error — subclasses must override.
//
// Ported from: packages/core/src/integration/integration.ts — getApiClient()
func (i *Integration) GetAPIClient() (any, error) {
	return nil, fmt.Errorf("method not implemented")
}
