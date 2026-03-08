// Ported from: packages/openai/src/tool/shell.ts
package openai

// ShellInputAction represents the action for the shell tool.
type ShellInputAction struct {
	// Commands is a list of shell commands to execute.
	Commands []string `json:"commands"`

	// TimeoutMs is an optional timeout in milliseconds for the commands.
	TimeoutMs *int `json:"timeoutMs,omitempty"`

	// MaxOutputLength is an optional maximum number of characters to return.
	MaxOutputLength *int `json:"maxOutputLength,omitempty"`
}

// ShellInput is the input schema for the shell tool.
type ShellInput struct {
	// Action contains the shell commands to execute.
	Action ShellInputAction `json:"action"`
}

// ShellOutputOutcome represents the outcome of a shell execution.
type ShellOutputOutcome struct {
	// Type is either "timeout" or "exit".
	Type string `json:"type"`

	// ExitCode is the exit code (only when Type is "exit").
	ExitCode *int `json:"exitCode,omitempty"`
}

// ShellOutputItem represents a single command output.
type ShellOutputItem struct {
	// Stdout is the standard output from the command.
	Stdout string `json:"stdout"`

	// Stderr is the standard error from the command.
	Stderr string `json:"stderr"`

	// Outcome is the outcome of the shell execution.
	Outcome ShellOutputOutcome `json:"outcome"`
}

// ShellOutput is the output schema for the shell tool.
type ShellOutput struct {
	// Output is an array of shell call output contents.
	Output []ShellOutputItem `json:"output"`
}

// ShellSkillReference represents a skill reference for the shell environment.
type ShellSkillReference struct {
	// Type is "skillReference".
	Type string `json:"type"`

	// SkillID is the skill identifier.
	SkillID string `json:"skillId"`

	// Version is an optional skill version.
	Version string `json:"version,omitempty"`
}

// ShellSkillInline represents an inline skill for the shell environment.
type ShellSkillInline struct {
	// Type is "inline".
	Type string `json:"type"`

	// Name is the skill name.
	Name string `json:"name"`

	// Description is the skill description.
	Description string `json:"description"`

	// Source is the skill source data.
	Source ShellSkillSource `json:"source"`
}

// ShellSkillSource is the source data for an inline skill.
type ShellSkillSource struct {
	// Type is "base64".
	Type string `json:"type"`

	// MediaType is "application/zip".
	MediaType string `json:"mediaType"`

	// Data is the base64-encoded skill data.
	Data string `json:"data"`
}

// ShellDomainSecret represents a domain secret for network allowlists.
type ShellDomainSecret struct {
	Domain string `json:"domain"`
	Name   string `json:"name"`
	Value  string `json:"value"`
}

// ShellNetworkPolicyDisabled represents a disabled network policy.
type ShellNetworkPolicyDisabled struct {
	Type string `json:"type"` // "disabled"
}

// ShellNetworkPolicyAllowlist represents a network allowlist policy.
type ShellNetworkPolicyAllowlist struct {
	Type           string              `json:"type"` // "allowlist"
	AllowedDomains []string            `json:"allowedDomains"`
	DomainSecrets  []ShellDomainSecret `json:"domainSecrets,omitempty"`
}

// ShellEnvironmentContainerAuto represents an auto-provisioned container environment.
type ShellEnvironmentContainerAuto struct {
	Type          string      `json:"type"` // "containerAuto"
	FileIDs       []string    `json:"fileIds,omitempty"`
	MemoryLimit   string      `json:"memoryLimit,omitempty"` // "1g", "4g", "16g", "64g"
	NetworkPolicy interface{} `json:"networkPolicy,omitempty"`
	Skills        interface{} `json:"skills,omitempty"`
}

// ShellEnvironmentContainerReference represents a reference to an existing container.
type ShellEnvironmentContainerReference struct {
	Type        string `json:"type"` // "containerReference"
	ContainerID string `json:"containerId"`
}

// ShellLocalSkill represents a local skill for the shell environment.
type ShellLocalSkill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

// ShellEnvironmentLocal represents a local shell environment.
type ShellEnvironmentLocal struct {
	Type   string            `json:"type,omitempty"` // "local"
	Skills []ShellLocalSkill `json:"skills,omitempty"`
}

// ShellArgs contains configuration options for the shell tool.
type ShellArgs struct {
	// Environment is the shell environment configuration.
	// Can be *ShellEnvironmentContainerAuto, *ShellEnvironmentContainerReference,
	// or *ShellEnvironmentLocal.
	Environment interface{} `json:"environment,omitempty"`
}

// ShellToolID is the provider tool ID for shell.
const ShellToolID = "openai.shell"

// NewShellTool creates a provider tool configuration for the shell tool.
func NewShellTool(args *ShellArgs) map[string]interface{} {
	result := map[string]interface{}{
		"type": "provider",
		"id":   ShellToolID,
	}
	if args != nil && args.Environment != nil {
		result["environment"] = args.Environment
	}
	return result
}
