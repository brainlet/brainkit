package workflow

import (
	"encoding/json"
	"time"
)

// AutomationManifest describes an automation package: workflow + admin code + deps.
type AutomationManifest struct {
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Type        string              `json:"type"` // "automation"
	Description string              `json:"description,omitempty"`
	Workflow    AutomationWorkflow  `json:"workflow"`
	Admin       string              `json:"admin,omitempty"` // admin.ts entry path
	Requires    *AutomationRequires `json:"requires,omitempty"`
}

// AutomationWorkflow describes the WASM workflow part of an automation.
type AutomationWorkflow struct {
	Entry    string           `json:"entry"`   // workflow.ts source file
	Triggers []TriggerDef     `json:"triggers,omitempty"`
	Timeout  int              `json:"timeout,omitempty"` // seconds, default 86400
	Retries  int              `json:"retries,omitempty"` // default 3
}

// AutomationRequires declares plugin and secret dependencies.
type AutomationRequires struct {
	Plugins      []string `json:"plugins,omitempty"`
	Secrets      []string `json:"secrets,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"` // e.g., "ai.generate"
}

// DeployedAutomation tracks a deployed automation.
type DeployedAutomation struct {
	Manifest   AutomationManifest `json:"manifest"`
	WorkflowID string             `json:"workflowId"`
	AdminSource string            `json:"adminSource,omitempty"`
	DeployedAt time.Time          `json:"deployedAt"`
	Status     string             `json:"status"` // "active", "stopped"
}

// ParseAutomationManifest parses a JSON automation manifest.
func ParseAutomationManifest(data []byte) (*AutomationManifest, error) {
	var m AutomationManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Type == "" {
		m.Type = "automation"
	}
	return &m, nil
}
