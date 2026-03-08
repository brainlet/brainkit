// Ported from: packages/core/src/tool-loop-agent/utils.ts
package toolloopagent

import (
	"errors"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// ToolLoopAgent is a stub for the AI SDK v6 ToolLoopAgent class.
// ai-kit only ported the @ai-sdk/provider layer (low-level interfaces).
// ToolLoopAgent is from @ai-sdk/ai (high-level orchestration), NOT ported.
type ToolLoopAgent struct {
	// Version identifies the agent version (e.g. "agent-v1").
	Version string `json:"version,omitempty"`

	// ID is the optional agent identifier.
	ID string `json:"id,omitempty"`

	// Tools holds the agent's tool set (map of name -> tool definition).
	Tools map[string]any `json:"tools,omitempty"`

	// Settings holds the internal ToolLoopAgentSettings.
	// In TypeScript this is private but accessible at runtime.
	Settings *ToolLoopAgentSettings `json:"settings,omitempty"`
}

// ToolLoopAgentSettings mirrors the AI SDK v6 ToolLoopAgentSettings.
// ai-kit only ported the @ai-sdk/provider layer (low-level interfaces).
// ToolLoopAgentSettings is from @ai-sdk/ai (high-level orchestration), NOT ported.
type ToolLoopAgentSettings struct {
	ID               string         `json:"id,omitempty"`
	Model            any            `json:"model,omitempty"`
	Tools            map[string]any `json:"tools,omitempty"`
	Instructions     any            `json:"instructions,omitempty"`
	MaxRetries       *int           `json:"maxRetries,omitempty"`
	MaxOutputTokens  *int           `json:"maxOutputTokens,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	TopP             *float64       `json:"topP,omitempty"`
	TopK             *int           `json:"topK,omitempty"`
	PresencePenalty  *float64       `json:"presencePenalty,omitempty"`
	FrequencyPenalty *float64       `json:"frequencyPenalty,omitempty"`
	StopSequences    []string       `json:"stopSequences,omitempty"`
	Seed             *int           `json:"seed,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	ActiveTools      []string       `json:"activeTools,omitempty"`
	ProviderOptions  map[string]map[string]any `json:"providerOptions,omitempty"`

	// ToolChoice controls how tools are selected.
	ToolChoice any `json:"toolChoice,omitempty"`

	// Callback hooks (function fields, not serialised).

	// StopWhen is a callback that determines whether to stop execution.
	StopWhen any `json:"-"`

	// OnStepFinish is called after each step completes.
	OnStepFinish any `json:"-"`

	// OnFinish is called when execution completes.
	OnFinish any `json:"-"`

	// PrepareCall is called before the first LLM call to allow overrides.
	PrepareCall func(input any) (map[string]any, error) `json:"-"`

	// PrepareStep is called before each step to allow per-step overrides.
	PrepareStep func(input any) (map[string]any, error) `json:"-"`

	// Experimental options.
	ExperimentalTelemetry any `json:"experimental_telemetry,omitempty"`
	ExperimentalContext   any `json:"experimental_context,omitempty"`
	ExperimentalDownload  any `json:"experimental_download,omitempty"`
}

// ---------------------------------------------------------------------------
// ToolLoopAgentLike
// ---------------------------------------------------------------------------

// ToolLoopAgentLike is the shape of a ToolLoopAgent-like object for runtime extraction.
// We use this looser interface because Go does not have TypeScript's structural typing
// with private properties across different package declarations.
type ToolLoopAgentLike interface {
	// GetID returns the optional agent identifier.
	GetID() string
	// GetVersion returns the version string (e.g. "agent-v1").
	GetVersion() string
	// GetSettings returns the internal settings. In TypeScript this is private
	// but accessible at runtime; in Go we expose it via an interface method.
	GetSettings() *ToolLoopAgentSettings
	// GetTools returns the agent's tool set, if any.
	GetTools() map[string]any
}

// Ensure ToolLoopAgent satisfies ToolLoopAgentLike.
var _ ToolLoopAgentLike = (*ToolLoopAgent)(nil)

// GetID implements ToolLoopAgentLike.
func (a *ToolLoopAgent) GetID() string { return a.ID }

// GetVersion implements ToolLoopAgentLike.
func (a *ToolLoopAgent) GetVersion() string { return a.Version }

// GetSettings implements ToolLoopAgentLike.
func (a *ToolLoopAgent) GetSettings() *ToolLoopAgentSettings { return a.Settings }

// GetTools implements ToolLoopAgentLike.
func (a *ToolLoopAgent) GetTools() map[string]any { return a.Tools }

// ---------------------------------------------------------------------------
// IsToolLoopAgentLike
// ---------------------------------------------------------------------------

// IsToolLoopAgentLike checks whether obj satisfies the ToolLoopAgentLike interface
// either via concrete type assertion or by checking the version string pattern.
func IsToolLoopAgentLike(obj any) bool {
	if obj == nil {
		return false
	}

	// Direct interface check.
	if _, ok := obj.(ToolLoopAgentLike); ok {
		return true
	}

	// Duck-type check: look for a Version field matching "agent-v*".
	type versioned interface {
		GetVersion() string
	}
	if v, ok := obj.(versioned); ok {
		ver := v.GetVersion()
		return ver == "agent-v1" || (len(ver) > 8 && ver[:8] == "agent-v")
	}

	return false
}

// ---------------------------------------------------------------------------
// GetSettings
// ---------------------------------------------------------------------------

// GetSettings extracts the ToolLoopAgentSettings from a ToolLoopAgentLike.
// Returns an error if the settings cannot be extracted (e.g. incompatible version).
func GetSettings(agent ToolLoopAgentLike) (*ToolLoopAgentSettings, error) {
	settings := agent.GetSettings()
	if settings == nil {
		return nil, errors.New("could not extract settings from ToolLoopAgent. The agent may be from an incompatible version")
	}
	return settings, nil
}
