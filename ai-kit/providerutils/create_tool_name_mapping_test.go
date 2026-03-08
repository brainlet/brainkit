// Ported from: packages/provider-utils/src/create-tool-name-mapping.test.ts
package providerutils

import "testing"

func TestCreateToolNameMapping_ProviderTools(t *testing.T) {
	tools := []ProviderToolDefinition{
		{Type: "provider", ID: "anthropic.computer-use", Name: "custom-computer-tool"},
		{Type: "provider", ID: "openai.code-interpreter", Name: "custom-code-tool"},
	}
	providerToolNames := map[string]string{
		"anthropic.computer-use":   "computer_use",
		"openai.code-interpreter": "code_interpreter",
	}

	mapping := CreateToolNameMapping(CreateToolNameMappingOptions{
		Tools:             tools,
		ProviderToolNames: providerToolNames,
	})

	if got := mapping.ToProviderToolName("custom-computer-tool"); got != "computer_use" {
		t.Errorf("expected 'computer_use', got %q", got)
	}
	if got := mapping.ToProviderToolName("custom-code-tool"); got != "code_interpreter" {
		t.Errorf("expected 'code_interpreter', got %q", got)
	}
	if got := mapping.ToCustomToolName("computer_use"); got != "custom-computer-tool" {
		t.Errorf("expected 'custom-computer-tool', got %q", got)
	}
	if got := mapping.ToCustomToolName("code_interpreter"); got != "custom-code-tool" {
		t.Errorf("expected 'custom-code-tool', got %q", got)
	}
}

func TestCreateToolNameMapping_IgnoresFunctionTools(t *testing.T) {
	tools := []ProviderToolDefinition{
		{Type: "function", Name: "my-function-tool"},
	}
	mapping := CreateToolNameMapping(CreateToolNameMappingOptions{
		Tools:             tools,
		ProviderToolNames: map[string]string{},
	})

	if got := mapping.ToProviderToolName("my-function-tool"); got != "my-function-tool" {
		t.Errorf("expected passthrough, got %q", got)
	}
}

func TestCreateToolNameMapping_UnknownToolPassthrough(t *testing.T) {
	tools := []ProviderToolDefinition{
		{Type: "provider", ID: "unknown.tool", Name: "custom-tool"},
	}
	mapping := CreateToolNameMapping(CreateToolNameMappingOptions{
		Tools:             tools,
		ProviderToolNames: map[string]string{},
	})

	if got := mapping.ToProviderToolName("custom-tool"); got != "custom-tool" {
		t.Errorf("expected passthrough, got %q", got)
	}
	if got := mapping.ToCustomToolName("unknown-name"); got != "unknown-name" {
		t.Errorf("expected passthrough, got %q", got)
	}
}

func TestCreateToolNameMapping_EmptyTools(t *testing.T) {
	mapping := CreateToolNameMapping(CreateToolNameMappingOptions{
		Tools:             nil,
		ProviderToolNames: map[string]string{},
	})

	if got := mapping.ToProviderToolName("any-tool"); got != "any-tool" {
		t.Errorf("expected passthrough, got %q", got)
	}
	if got := mapping.ToCustomToolName("any-tool"); got != "any-tool" {
		t.Errorf("expected passthrough, got %q", got)
	}
}

func TestCreateToolNameMapping_MixedTools(t *testing.T) {
	tools := []ProviderToolDefinition{
		{Type: "function", Name: "function-tool"},
		{Type: "provider", ID: "anthropic.computer-use", Name: "provider-tool"},
	}
	providerToolNames := map[string]string{
		"anthropic.computer-use": "computer_use",
	}

	mapping := CreateToolNameMapping(CreateToolNameMappingOptions{
		Tools:             tools,
		ProviderToolNames: providerToolNames,
	})

	// Function tool should not be mapped
	if got := mapping.ToProviderToolName("function-tool"); got != "function-tool" {
		t.Errorf("expected passthrough for function tool, got %q", got)
	}
	// Provider tool should be mapped
	if got := mapping.ToProviderToolName("provider-tool"); got != "computer_use" {
		t.Errorf("expected 'computer_use', got %q", got)
	}
}

func TestCreateToolNameMapping_DynamicResolver(t *testing.T) {
	tools := []ProviderToolDefinition{
		{Type: "provider", ID: "openai.custom", Name: "alias_name"},
	}

	mapping := CreateToolNameMapping(CreateToolNameMappingOptions{
		Tools:             tools,
		ProviderToolNames: map[string]string{},
		ResolveProviderToolName: func(tool ProviderToolDefinition) *string {
			if tool.ID == "openai.custom" {
				name := "write_sql"
				return &name
			}
			return nil
		},
	})

	if got := mapping.ToProviderToolName("alias_name"); got != "write_sql" {
		t.Errorf("expected 'write_sql', got %q", got)
	}
	if got := mapping.ToCustomToolName("write_sql"); got != "alias_name" {
		t.Errorf("expected 'alias_name', got %q", got)
	}
}
