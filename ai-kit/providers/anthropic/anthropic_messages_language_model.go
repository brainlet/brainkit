// Ported from: packages/anthropic/src/anthropic-messages-language-model.ts
package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// AnthropicMessagesConfig is the configuration for the Anthropic Messages language model.
type AnthropicMessagesConfig struct {
	Provider                      string
	BaseURL                       string
	Headers                       func() map[string]string
	Fetch                         providerutils.FetchFunction
	BuildRequestURL               func(baseURL string, isStreaming bool) string
	TransformRequestBody          func(args map[string]any, betas map[string]bool) map[string]any
	SupportedURLs                 func() map[string][]*regexp.Regexp
	GenerateID                    func() string
	SupportsNativeStructuredOutput *bool
}

// AnthropicMessagesLanguageModel implements the LanguageModel interface for Anthropic's Messages API.
type AnthropicMessagesLanguageModel struct {
	modelID    AnthropicMessagesModelId
	config     AnthropicMessagesConfig
	generateID func() string
}

// NewAnthropicMessagesLanguageModel creates a new AnthropicMessagesLanguageModel.
func NewAnthropicMessagesLanguageModel(
	modelID AnthropicMessagesModelId,
	config AnthropicMessagesConfig,
) *AnthropicMessagesLanguageModel {
	genID := config.GenerateID
	if genID == nil {
		genID = providerutils.GenerateId
	}
	return &AnthropicMessagesLanguageModel{
		modelID:    modelID,
		config:     config,
		generateID: genID,
	}
}

func (m *AnthropicMessagesLanguageModel) SpecificationVersion() string {
	return "v3"
}

func (m *AnthropicMessagesLanguageModel) Provider() string {
	return m.config.Provider
}

func (m *AnthropicMessagesLanguageModel) ModelID() string {
	return m.modelID
}

// providerOptionsName extracts the dynamic provider name from the config.Provider string.
// e.g., "my-custom-anthropic.messages" -> "my-custom-anthropic"
func (m *AnthropicMessagesLanguageModel) providerOptionsName() string {
	provider := m.config.Provider
	dotIndex := strings.Index(provider, ".")
	if dotIndex == -1 {
		return provider
	}
	return provider[:dotIndex]
}

func (m *AnthropicMessagesLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	if m.config.SupportedURLs != nil {
		return m.config.SupportedURLs(), nil
	}
	return map[string][]*regexp.Regexp{}, nil
}

// getArgsResult holds the result of getArgs.
type getArgsResult struct {
	args                 map[string]any
	warnings             []shared.Warning
	betas                map[string]bool
	usesJsonResponseTool bool
	toolNameMapping      providerutils.ToolNameMapping
	providerOptionsName  string
	usedCustomProviderKey bool
}

// getArgs builds the request arguments from CallOptions.
func (m *AnthropicMessagesLanguageModel) getArgs(
	options languagemodel.CallOptions,
	stream bool,
	userSuppliedBetas map[string]bool,
) (*getArgsResult, error) {
	var warnings []shared.Warning

	if options.FrequencyPenalty != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "frequencyPenalty"})
	}

	if options.PresencePenalty != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "presencePenalty"})
	}

	if options.Seed != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "seed"})
	}

	temperature := options.Temperature
	if temperature != nil && *temperature > 1 {
		details := fmt.Sprintf("%g exceeds anthropic maximum of 1.0. clamped to 1.0", *temperature)
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "temperature", Details: &details})
		clamped := 1.0
		temperature = &clamped
	} else if temperature != nil && *temperature < 0 {
		details := fmt.Sprintf("%g is below anthropic minimum of 0. clamped to 0", *temperature)
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "temperature", Details: &details})
		clamped := 0.0
		temperature = &clamped
	}

	if rf, ok := options.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
		if rf.Schema == nil {
			details := "JSON response format requires a schema. The response format is ignored."
			warnings = append(warnings, shared.UnsupportedWarning{Feature: "responseFormat", Details: &details})
		}
	}

	providerOptionsName := m.providerOptionsName()

	// Parse provider options from both canonical 'anthropic' key and custom key
	var anthropicOptions *AnthropicLanguageModelOptions
	if options.ProviderOptions != nil {
		if aoRaw, ok := options.ProviderOptions["anthropic"]; ok {
			anthropicOptions = parseAnthropicOptions(aoRaw)
		}
	}

	var customProviderOptions *AnthropicLanguageModelOptions
	usedCustomProviderKey := false
	if providerOptionsName != "anthropic" && options.ProviderOptions != nil {
		if cpRaw, ok := options.ProviderOptions[providerOptionsName]; ok {
			customProviderOptions = parseAnthropicOptions(cpRaw)
			usedCustomProviderKey = true
		}
	}

	// Merge options
	if customProviderOptions != nil {
		if anthropicOptions == nil {
			anthropicOptions = customProviderOptions
		} else {
			mergeAnthropicOptions(anthropicOptions, customProviderOptions)
		}
	}
	if anthropicOptions == nil {
		anthropicOptions = &AnthropicLanguageModelOptions{}
	}

	caps := getModelCapabilities(m.modelID)

	supportsNativeStructuredOutput := true
	if m.config.SupportsNativeStructuredOutput != nil {
		supportsNativeStructuredOutput = *m.config.SupportsNativeStructuredOutput
	}
	supportsStructuredOutput := supportsNativeStructuredOutput && caps.supportsStructuredOutput

	structuredOutputMode := "auto"
	if anthropicOptions.StructuredOutputMode != nil {
		structuredOutputMode = *anthropicOptions.StructuredOutputMode
	}
	useStructuredOutput := structuredOutputMode == "outputFormat" ||
		(structuredOutputMode == "auto" && supportsStructuredOutput)

	// JSON response tool fallback
	var jsonResponseTool *languagemodel.FunctionTool
	if rf, ok := options.ResponseFormat.(languagemodel.ResponseFormatJSON); ok && rf.Schema != nil && !useStructuredOutput {
		jsonResponseTool = &languagemodel.FunctionTool{
			Name:        "json",
			Description: strPtr("Respond with a JSON object."),
			InputSchema: rf.Schema,
		}
	}

	contextManagement := anthropicOptions.ContextManagement

	// Create a shared cache control validator
	cacheControlValidator := NewCacheControlValidator()

	// Build tool name mapping
	toolDefs := make([]providerutils.ProviderToolDefinition, 0, len(options.Tools))
	for _, tool := range options.Tools {
		switch t := tool.(type) {
		case languagemodel.FunctionTool:
			toolDefs = append(toolDefs, providerutils.ProviderToolDefinition{
				Type: "function",
				Name: t.Name,
			})
		case languagemodel.ProviderTool:
			toolDefs = append(toolDefs, providerutils.ProviderToolDefinition{
				Type: "provider",
				Name: t.Name,
				ID:   t.ID,
			})
		}
	}

	toolNameMapping := providerutils.CreateToolNameMapping(providerutils.CreateToolNameMappingOptions{
		Tools: toolDefs,
		ProviderToolNames: map[string]string{
			"anthropic.code_execution_20250522": "code_execution",
			"anthropic.code_execution_20250825": "code_execution",
			"anthropic.code_execution_20260120": "code_execution",
			"anthropic.computer_20241022":       "computer",
			"anthropic.computer_20250124":       "computer",
			"anthropic.text_editor_20241022":    "str_replace_editor",
			"anthropic.text_editor_20250124":    "str_replace_editor",
			"anthropic.text_editor_20250429":    "str_replace_based_edit_tool",
			"anthropic.text_editor_20250728":    "str_replace_based_edit_tool",
			"anthropic.bash_20241022":           "bash",
			"anthropic.bash_20250124":           "bash",
			"anthropic.memory_20250818":         "memory",
			"anthropic.web_search_20250305":     "web_search",
			"anthropic.web_search_20260209":     "web_search",
			"anthropic.web_fetch_20250910":      "web_fetch",
			"anthropic.web_fetch_20260209":      "web_fetch",
			"anthropic.tool_search_regex_20251119": "tool_search_tool_regex",
			"anthropic.tool_search_bm25_20251119":  "tool_search_tool_bm25",
		},
	})

	// Convert prompt
	sendReasoning := true
	if anthropicOptions.SendReasoning != nil {
		sendReasoning = *anthropicOptions.SendReasoning
	}

	betas := map[string]bool{}
	promptResult := convertToAnthropicMessagesPrompt(
		options.Prompt,
		sendReasoning,
		&warnings,
		cacheControlValidator,
	)
	for k, v := range promptResult.Betas {
		betas[k] = v
	}

	thinkingType := ""
	if anthropicOptions.Thinking != nil {
		thinkingType = anthropicOptions.Thinking.Type
	}
	isThinking := thinkingType == "enabled" || thinkingType == "adaptive"
	var thinkingBudget *int
	if thinkingType == "enabled" && anthropicOptions.Thinking != nil {
		thinkingBudget = anthropicOptions.Thinking.BudgetTokens
	}

	maxTokens := 0
	if options.MaxOutputTokens != nil {
		maxTokens = *options.MaxOutputTokens
	} else {
		maxTokens = caps.maxOutputTokens
	}

	// Build base args
	baseArgs := map[string]any{
		"model":      m.modelID,
		"max_tokens": maxTokens,
	}
	if temperature != nil {
		baseArgs["temperature"] = *temperature
	}
	if options.TopK != nil {
		baseArgs["top_k"] = *options.TopK
	}
	if options.TopP != nil {
		baseArgs["top_p"] = *options.TopP
	}
	if len(options.StopSequences) > 0 {
		baseArgs["stop_sequences"] = options.StopSequences
	}

	// Thinking
	if isThinking {
		thinking := map[string]any{
			"type": thinkingType,
		}
		if thinkingBudget != nil {
			thinking["budget_tokens"] = *thinkingBudget
		}
		baseArgs["thinking"] = thinking
	}

	// Output config (effort + structured output)
	needsOutputConfig := anthropicOptions.Effort != nil ||
		(useStructuredOutput && isJsonFormatWithSchema(options.ResponseFormat))
	if needsOutputConfig {
		outputConfig := map[string]any{}
		if anthropicOptions.Effort != nil {
			outputConfig["effort"] = *anthropicOptions.Effort
		}
		if useStructuredOutput {
			if rf, ok := options.ResponseFormat.(languagemodel.ResponseFormatJSON); ok && rf.Schema != nil {
				outputConfig["format"] = map[string]any{
					"type":   "json_schema",
					"schema": rf.Schema,
				}
			}
		}
		baseArgs["output_config"] = outputConfig
	}

	// Speed
	if anthropicOptions.Speed != nil {
		baseArgs["speed"] = *anthropicOptions.Speed
	}

	// Cache control
	if anthropicOptions.CacheControl != nil {
		baseArgs["cache_control"] = anthropicOptions.CacheControl
	}

	// MCP servers
	if len(anthropicOptions.MCPServers) > 0 {
		mcpServers := make([]map[string]any, 0, len(anthropicOptions.MCPServers))
		for _, server := range anthropicOptions.MCPServers {
			s := map[string]any{
				"type": server.Type,
				"name": server.Name,
				"url":  server.URL,
			}
			if server.AuthorizationToken != nil {
				s["authorization_token"] = *server.AuthorizationToken
			}
			if server.ToolConfiguration != nil {
				tc := map[string]any{}
				if server.ToolConfiguration.AllowedTools != nil {
					tc["allowed_tools"] = server.ToolConfiguration.AllowedTools
				}
				if server.ToolConfiguration.Enabled != nil {
					tc["enabled"] = *server.ToolConfiguration.Enabled
				}
				s["tool_configuration"] = tc
			}
			mcpServers = append(mcpServers, s)
		}
		baseArgs["mcp_servers"] = mcpServers
	}

	// Container
	if anthropicOptions.Container != nil {
		if len(anthropicOptions.Container.Skills) > 0 {
			skills := make([]map[string]any, 0, len(anthropicOptions.Container.Skills))
			for _, skill := range anthropicOptions.Container.Skills {
				sk := map[string]any{
					"type":     skill.Type,
					"skill_id": skill.SkillID,
				}
				if skill.Version != nil {
					sk["version"] = *skill.Version
				}
				skills = append(skills, sk)
			}
			container := map[string]any{}
			if anthropicOptions.Container.ID != nil {
				container["id"] = *anthropicOptions.Container.ID
			}
			container["skills"] = skills
			baseArgs["container"] = container
		} else if anthropicOptions.Container.ID != nil {
			baseArgs["container"] = *anthropicOptions.Container.ID
		}
	}

	// Prompt
	baseArgs["system"] = promptResult.Prompt.System
	baseArgs["messages"] = promptResult.Prompt.Messages

	// Context management
	if contextManagement != nil {
		edits := make([]map[string]any, 0, len(contextManagement.Edits))
		for _, edit := range contextManagement.Edits {
			switch edit.Type {
			case "clear_tool_uses_20250919":
				e := map[string]any{"type": edit.Type}
				if edit.Trigger != nil {
					e["trigger"] = map[string]any{
						"type":  edit.Trigger.Type,
						"value": edit.Trigger.Value,
					}
				}
				if edit.Keep != nil {
					e["keep"] = map[string]any{
						"type":  edit.Keep.Type,
						"value": edit.Keep.Value,
					}
				}
				if edit.ClearAtLeast != nil {
					e["clear_at_least"] = map[string]any{
						"type":  edit.ClearAtLeast.Type,
						"value": edit.ClearAtLeast.Value,
					}
				}
				if edit.ClearToolInputs != nil {
					e["clear_tool_inputs"] = *edit.ClearToolInputs
				}
				if len(edit.ExcludeTools) > 0 {
					e["exclude_tools"] = edit.ExcludeTools
				}
				edits = append(edits, e)

			case "clear_thinking_20251015":
				e := map[string]any{"type": edit.Type}
				if edit.Keep != nil {
					e["keep"] = map[string]any{
						"type":  edit.Keep.Type,
						"value": edit.Keep.Value,
					}
				} else if edit.KeepAll != nil && *edit.KeepAll {
					e["keep"] = "all"
				}
				edits = append(edits, e)

			case "compact_20260112":
				e := map[string]any{"type": edit.Type}
				if edit.Trigger != nil {
					e["trigger"] = map[string]any{
						"type":  edit.Trigger.Type,
						"value": edit.Trigger.Value,
					}
				}
				if edit.PauseAfterCompaction != nil {
					e["pause_after_compaction"] = *edit.PauseAfterCompaction
				}
				if edit.Instructions != nil {
					e["instructions"] = *edit.Instructions
				}
				edits = append(edits, e)

			default:
				msg := fmt.Sprintf("Unknown context management strategy: %s", edit.Type)
				warnings = append(warnings, shared.OtherWarning{Message: msg})
			}
		}
		baseArgs["context_management"] = map[string]any{
			"edits": edits,
		}
	}

	// Thinking adjustments
	if isThinking {
		if thinkingType == "enabled" && thinkingBudget == nil {
			details := "thinking budget is required when thinking is enabled. using default budget of 1024 tokens."
			warnings = append(warnings, shared.CompatibilityWarning{Feature: "extended thinking", Details: &details})
			baseArgs["thinking"] = map[string]any{
				"type":          "enabled",
				"budget_tokens": 1024,
			}
			defaultBudget := 1024
			thinkingBudget = &defaultBudget
		}

		if _, ok := baseArgs["temperature"]; ok {
			delete(baseArgs, "temperature")
			details := "temperature is not supported when thinking is enabled"
			warnings = append(warnings, shared.UnsupportedWarning{Feature: "temperature", Details: &details})
		}

		if _, ok := baseArgs["top_k"]; ok {
			delete(baseArgs, "top_k")
			details := "topK is not supported when thinking is enabled"
			warnings = append(warnings, shared.UnsupportedWarning{Feature: "topK", Details: &details})
		}

		if _, ok := baseArgs["top_p"]; ok {
			delete(baseArgs, "top_p")
			details := "topP is not supported when thinking is enabled"
			warnings = append(warnings, shared.UnsupportedWarning{Feature: "topP", Details: &details})
		}

		// Adjust max tokens to account for thinking
		budget := 0
		if thinkingBudget != nil {
			budget = *thinkingBudget
		}
		baseArgs["max_tokens"] = maxTokens + budget
	} else {
		// Only check temperature/topP mutual exclusivity when thinking is not enabled
		if options.TopP != nil && temperature != nil {
			delete(baseArgs, "top_p")
			details := "topP is not supported when temperature is set. topP is ignored."
			warnings = append(warnings, shared.UnsupportedWarning{Feature: "topP", Details: &details})
		}
	}

	// Limit to max output tokens for known models
	if caps.isKnownModel {
		currentMaxTokens, _ := baseArgs["max_tokens"].(int)
		if currentMaxTokens > caps.maxOutputTokens {
			if options.MaxOutputTokens != nil {
				details := fmt.Sprintf(
					"%d (maxOutputTokens + thinkingBudget) is greater than %s %d max output tokens. The max output tokens have been limited to %d.",
					currentMaxTokens, m.modelID, caps.maxOutputTokens, caps.maxOutputTokens,
				)
				warnings = append(warnings, shared.UnsupportedWarning{Feature: "maxOutputTokens", Details: &details})
			}
			baseArgs["max_tokens"] = caps.maxOutputTokens
		}
	}

	// MCP betas
	if len(anthropicOptions.MCPServers) > 0 {
		betas["mcp-client-2025-04-04"] = true
	}

	// Context management betas
	if contextManagement != nil {
		betas["context-management-2025-06-27"] = true
		for _, e := range contextManagement.Edits {
			if e.Type == "compact_20260112" {
				betas["compact-2026-01-12"] = true
				break
			}
		}
	}

	// Container skills betas
	if anthropicOptions.Container != nil && len(anthropicOptions.Container.Skills) > 0 {
		betas["code-execution-2025-08-25"] = true
		betas["skills-2025-10-02"] = true
		betas["files-api-2025-04-14"] = true

		hasCodeExecTool := false
		for _, tool := range options.Tools {
			if pt, ok := tool.(languagemodel.ProviderTool); ok {
				if pt.ID == "anthropic.code_execution_20250825" || pt.ID == "anthropic.code_execution_20260120" {
					hasCodeExecTool = true
					break
				}
			}
		}
		if !hasCodeExecTool {
			warnings = append(warnings, shared.OtherWarning{Message: "code execution tool is required when using skills"})
		}
	}

	// Effort beta
	if anthropicOptions.Effort != nil {
		betas["effort-2025-11-24"] = true
	}

	// Speed beta
	if anthropicOptions.Speed != nil && *anthropicOptions.Speed == "fast" {
		betas["fast-mode-2026-02-01"] = true
	}

	// Tool streaming beta (only when streaming)
	if stream {
		toolStreaming := true
		if anthropicOptions.ToolStreaming != nil {
			toolStreaming = *anthropicOptions.ToolStreaming
		}
		if toolStreaming {
			betas["fine-grained-tool-streaming-2025-05-14"] = true
		}
	}

	// Prepare tools
	var toolsInput []languagemodel.Tool
	var toolChoiceInput languagemodel.ToolChoice
	var disableParallelToolUse *bool
	var prepareSupportsStructuredOutput bool

	if jsonResponseTool != nil {
		toolsInput = make([]languagemodel.Tool, 0, len(options.Tools)+1)
		toolsInput = append(toolsInput, options.Tools...)
		toolsInput = append(toolsInput, *jsonResponseTool)
		toolChoiceInput = languagemodel.ToolChoiceRequired{}
		t := true
		disableParallelToolUse = &t
		prepareSupportsStructuredOutput = false
	} else {
		toolsInput = options.Tools
		toolChoiceInput = options.ToolChoice
		if anthropicOptions.DisableParallelToolUse != nil {
			disableParallelToolUse = anthropicOptions.DisableParallelToolUse
		}
		prepareSupportsStructuredOutput = supportsStructuredOutput
	}

	toolsResult := prepareTools(
		toolsInput,
		toolChoiceInput,
		disableParallelToolUse,
		cacheControlValidator,
		prepareSupportsStructuredOutput,
	)

	// Cache control warnings
	cacheWarnings := cacheControlValidator.GetWarnings()

	// Build final args
	if toolsResult.Tools != nil {
		baseArgs["tools"] = toolsResult.Tools
	}
	if toolsResult.ToolChoice != nil {
		baseArgs["tool_choice"] = toolsResult.ToolChoice
	}
	if stream {
		baseArgs["stream"] = true
	}

	// Merge all warnings
	allWarnings := make([]shared.Warning, 0, len(warnings)+len(toolsResult.ToolWarnings)+len(cacheWarnings))
	allWarnings = append(allWarnings, warnings...)
	allWarnings = append(allWarnings, toolsResult.ToolWarnings...)
	allWarnings = append(allWarnings, cacheWarnings...)

	// Merge all betas
	for k, v := range toolsResult.Betas {
		betas[k] = v
	}
	for k, v := range userSuppliedBetas {
		betas[k] = v
	}
	for _, b := range anthropicOptions.AnthropicBeta {
		betas[b] = true
	}

	return &getArgsResult{
		args:                  baseArgs,
		warnings:             allWarnings,
		betas:                betas,
		usesJsonResponseTool: jsonResponseTool != nil,
		toolNameMapping:      toolNameMapping,
		providerOptionsName:  providerOptionsName,
		usedCustomProviderKey: usedCustomProviderKey,
	}, nil
}

// getHeaders builds the request headers including beta headers.
func (m *AnthropicMessagesLanguageModel) getHeaders(betas map[string]bool, optHeaders map[string]*string) map[string]string {
	configHeaders := m.config.Headers()

	headers := providerutils.CombineHeaders(configHeaders)

	// Add option headers
	if optHeaders != nil {
		for k, v := range optHeaders {
			if v != nil {
				headers[k] = *v
			}
		}
	}

	// Add beta header
	if len(betas) > 0 {
		betaList := make([]string, 0, len(betas))
		for b := range betas {
			betaList = append(betaList, b)
		}
		headers["anthropic-beta"] = strings.Join(betaList, ",")
	}

	return headers
}

// getBetasFromHeaders extracts beta values from config and request headers.
func (m *AnthropicMessagesLanguageModel) getBetasFromHeaders(requestHeaders map[string]*string) map[string]bool {
	configHeaders := m.config.Headers()
	result := map[string]bool{}

	configBetaHeader := configHeaders["anthropic-beta"]
	if configBetaHeader != "" {
		for _, b := range strings.Split(strings.ToLower(configBetaHeader), ",") {
			b = strings.TrimSpace(b)
			if b != "" {
				result[b] = true
			}
		}
	}

	if requestHeaders != nil {
		if reqBeta, ok := requestHeaders["anthropic-beta"]; ok && reqBeta != nil {
			for _, b := range strings.Split(strings.ToLower(*reqBeta), ",") {
				b = strings.TrimSpace(b)
				if b != "" {
					result[b] = true
				}
			}
		}
	}

	return result
}

func (m *AnthropicMessagesLanguageModel) buildRequestURL(isStreaming bool) string {
	if m.config.BuildRequestURL != nil {
		return m.config.BuildRequestURL(m.config.BaseURL, isStreaming)
	}
	return m.config.BaseURL + "/messages"
}

func (m *AnthropicMessagesLanguageModel) transformRequestBody(args map[string]any, betas map[string]bool) map[string]any {
	if m.config.TransformRequestBody != nil {
		return m.config.TransformRequestBody(args, betas)
	}
	return args
}

// extractCitationDocuments extracts citation document info from the prompt.
func (m *AnthropicMessagesLanguageModel) extractCitationDocuments(prompt languagemodel.Prompt) []citationDocumentInfo {
	var docs []citationDocumentInfo
	for _, msg := range prompt {
		userMsg, ok := msg.(languagemodel.UserMessage)
		if !ok {
			continue
		}
		for _, part := range userMsg.Content {
			fp, ok := part.(languagemodel.FilePart)
			if !ok {
				continue
			}
			if fp.MediaType != "application/pdf" && fp.MediaType != "text/plain" {
				continue
			}
			if fp.ProviderOptions == nil {
				continue
			}
			antOpts, ok := fp.ProviderOptions["anthropic"]
			if !ok || antOpts == nil {
				continue
			}
			citRaw, ok := antOpts["citations"]
			if !ok {
				continue
			}
			citMap, ok := citRaw.(map[string]any)
			if !ok {
				continue
			}
			enabled, ok := citMap["enabled"].(bool)
			if !ok || !enabled {
				continue
			}
			title := "Untitled Document"
			if fp.Filename != nil {
				title = *fp.Filename
			}
			doc := citationDocumentInfo{
				Title:     title,
				MediaType: fp.MediaType,
			}
			if fp.Filename != nil {
				doc.Filename = fp.Filename
			}
			docs = append(docs, doc)
		}
	}
	return docs
}

type citationDocumentInfo struct {
	Title     string
	Filename  *string
	MediaType string
}

// DoGenerate performs a non-streaming request to the Anthropic Messages API.
func (m *AnthropicMessagesLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	userBetas := m.getBetasFromHeaders(options.Headers)

	prepared, err := m.getArgs(options, false, userBetas)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	citationDocuments := m.extractCitationDocuments(options.Prompt)
	markCodeExecutionDynamic := hasWebTool20260209WithoutCodeExecution(prepared.args)

	ctx := options.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[AnthropicMessagesResponse]{
		URL:     m.buildRequestURL(false),
		Headers: m.getHeaders(prepared.betas, options.Headers),
		Body:    m.transformRequestBody(prepared.args, prepared.betas),
		FailedResponseHandler:     wrapErrorResponseHandler(anthropicFailedResponseHandler),
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(&providerutils.Schema[AnthropicMessagesResponse]{}),
		Ctx:   ctx,
		Fetch: m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	response := result.Value
	var content []languagemodel.Content
	mcpToolCalls := map[string]languagemodel.ToolCall{}
	serverToolCalls := map[string]string{}
	isJsonResponseFromTool := false

	for _, part := range response.Content {
		partType, _ := part["type"].(string)

		switch partType {
		case "text":
			text, _ := part["text"].(string)
			if !prepared.usesJsonResponseTool {
				content = append(content, languagemodel.Text{Text: text})

				// Process citations
				if citations, ok := part["citations"].([]any); ok {
					for _, citRaw := range citations {
						citMap, ok := citRaw.(map[string]any)
						if !ok {
							continue
						}
						source := createCitationSource(citMap, citationDocuments, m.generateID)
						if source != nil {
							content = append(content, source)
						}
					}
				}
			}

		case "thinking":
			thinking, _ := part["thinking"].(string)
			signature, _ := part["signature"].(string)
			content = append(content, languagemodel.Reasoning{
				Text: thinking,
				ProviderMetadata: shared.ProviderMetadata{
					"anthropic": jsonvalue.JSONObject{
						"signature": signature,
					},
				},
			})

		case "redacted_thinking":
			data, _ := part["data"].(string)
			content = append(content, languagemodel.Reasoning{
				Text: "",
				ProviderMetadata: shared.ProviderMetadata{
					"anthropic": jsonvalue.JSONObject{
						"redactedData": data,
					},
				},
			})

		case "compaction":
			compactionContent, _ := part["content"].(string)
			content = append(content, languagemodel.Text{
				Text: compactionContent,
				ProviderMetadata: shared.ProviderMetadata{
					"anthropic": jsonvalue.JSONObject{
						"type": "compaction",
					},
				},
			})

		case "tool_use":
			toolName, _ := part["name"].(string)
			toolID, _ := part["id"].(string)
			input := part["input"]

			if prepared.usesJsonResponseTool && toolName == "json" {
				isJsonResponseFromTool = true
				inputJSON, _ := json.Marshal(input)
				content = append(content, languagemodel.Text{Text: string(inputJSON)})
			} else {
				inputJSON, _ := json.Marshal(input)
				tc := languagemodel.ToolCall{
					ToolCallID: toolID,
					ToolName:   toolName,
					Input:      string(inputJSON),
				}
				if callerRaw, ok := part["caller"]; ok && callerRaw != nil {
					if callerMap, ok := callerRaw.(map[string]any); ok {
						callerType, _ := callerMap["type"].(string)
						callerInfo := map[string]any{"type": callerType}
						if toolIDVal, ok := callerMap["tool_id"].(string); ok {
							callerInfo["toolId"] = toolIDVal
						}
						tc.ProviderMetadata = shared.ProviderMetadata{
							"anthropic": jsonvalue.JSONObject{
								"caller": callerInfo,
							},
						}
					}
				}
				content = append(content, tc)
			}

		case "server_tool_use":
			toolName, _ := part["name"].(string)
			toolID, _ := part["id"].(string)
			input := part["input"]

			if toolName == "text_editor_code_execution" || toolName == "bash_code_execution" {
				inputMap := map[string]any{"type": toolName}
				if m2, ok := input.(map[string]any); ok {
					for k, v := range m2 {
						inputMap[k] = v
					}
				}
				inputJSON, _ := json.Marshal(inputMap)
				pe := true
				content = append(content, languagemodel.ToolCall{
					ToolCallID:       toolID,
					ToolName:         prepared.toolNameMapping.ToCustomToolName("code_execution"),
					Input:            string(inputJSON),
					ProviderExecuted: &pe,
				})
			} else if toolName == "web_search" || toolName == "code_execution" || toolName == "web_fetch" {
				inputToSerialize := input
				if toolName == "code_execution" {
					if m2, ok := input.(map[string]any); ok {
						if _, hasCode := m2["code"]; hasCode {
							if _, hasType := m2["type"]; !hasType {
								newInput := map[string]any{"type": "programmatic-tool-call"}
								for k, v := range m2 {
									newInput[k] = v
								}
								inputToSerialize = newInput
							}
						}
					}
				}
				inputJSON, _ := json.Marshal(inputToSerialize)
				pe := true
				tc := languagemodel.ToolCall{
					ToolCallID:       toolID,
					ToolName:         prepared.toolNameMapping.ToCustomToolName(toolName),
					Input:            string(inputJSON),
					ProviderExecuted: &pe,
				}
				if markCodeExecutionDynamic && toolName == "code_execution" {
					d := true
					tc.Dynamic = &d
				}
				content = append(content, tc)
			} else if toolName == "tool_search_tool_regex" || toolName == "tool_search_tool_bm25" {
				serverToolCalls[toolID] = toolName
				inputJSON, _ := json.Marshal(input)
				pe := true
				content = append(content, languagemodel.ToolCall{
					ToolCallID:       toolID,
					ToolName:         prepared.toolNameMapping.ToCustomToolName(toolName),
					Input:            string(inputJSON),
					ProviderExecuted: &pe,
				})
			}

		case "mcp_tool_use":
			toolName, _ := part["name"].(string)
			toolID, _ := part["id"].(string)
			serverName, _ := part["server_name"].(string)
			input := part["input"]
			inputJSON, _ := json.Marshal(input)
			pe := true
			d := true
			tc := languagemodel.ToolCall{
				ToolCallID:       toolID,
				ToolName:         toolName,
				Input:            string(inputJSON),
				ProviderExecuted: &pe,
				Dynamic:          &d,
				ProviderMetadata: shared.ProviderMetadata{
					"anthropic": jsonvalue.JSONObject{
						"type":       "mcp-tool-use",
						"serverName": serverName,
					},
				},
			}
			mcpToolCalls[toolID] = tc
			content = append(content, tc)

		case "mcp_tool_result":
			toolUseID, _ := part["tool_use_id"].(string)
			isError, _ := part["is_error"].(bool)
			resultContent := part["content"]
			mcpTC := mcpToolCalls[toolUseID]
			d := true
			content = append(content, languagemodel.ToolResult{
				ToolCallID:       toolUseID,
				ToolName:         mcpTC.ToolName,
				IsError:          &isError,
				Result:           resultContent,
				Dynamic:          &d,
				ProviderMetadata: mcpTC.ProviderMetadata,
			})

		case "web_fetch_tool_result":
			toolUseID, _ := part["tool_use_id"].(string)
			contentMap, _ := part["content"].(map[string]any)
			contentType2, _ := contentMap["type"].(string)

			if contentType2 == "web_fetch_result" {
				// Add to citation documents
				innerContent, _ := contentMap["content"].(map[string]any)
				fetchTitle := ""
				if t, ok := innerContent["title"].(string); ok {
					fetchTitle = t
				} else if u, ok := contentMap["url"].(string); ok {
					fetchTitle = u
				}
				source, _ := innerContent["source"].(map[string]any)
				mediaType, _ := source["media_type"].(string)
				citationDocuments = append(citationDocuments, citationDocumentInfo{
					Title:     fetchTitle,
					MediaType: mediaType,
				})

				content = append(content, languagemodel.ToolResult{
					ToolCallID: toolUseID,
					ToolName:   prepared.toolNameMapping.ToCustomToolName("web_fetch"),
					Result:     buildWebFetchResult(contentMap),
				})
			} else if contentType2 == "web_fetch_tool_result_error" {
				isErr := true
				errorCode, _ := contentMap["error_code"].(string)
				content = append(content, languagemodel.ToolResult{
					ToolCallID: toolUseID,
					ToolName:   prepared.toolNameMapping.ToCustomToolName("web_fetch"),
					IsError:    &isErr,
					Result: map[string]any{
						"type":      "web_fetch_tool_result_error",
						"errorCode": errorCode,
					},
				})
			}

		case "web_search_tool_result":
			toolUseID, _ := part["tool_use_id"].(string)
			contentRaw := part["content"]

			if contentArr, ok := contentRaw.([]any); ok {
				results := make([]map[string]any, 0, len(contentArr))
				for _, r := range contentArr {
					if rm, ok := r.(map[string]any); ok {
						result := map[string]any{
							"url":              rm["url"],
							"title":            rm["title"],
							"type":             rm["type"],
							"encryptedContent": rm["encrypted_content"],
						}
						if pa, ok := rm["page_age"]; ok {
							result["pageAge"] = pa
						} else {
							result["pageAge"] = nil
						}
						results = append(results, result)
					}
				}
				content = append(content, languagemodel.ToolResult{
					ToolCallID: toolUseID,
					ToolName:   prepared.toolNameMapping.ToCustomToolName("web_search"),
					Result:     results,
				})
				// Add sources
				for _, r := range contentArr {
					if rm, ok := r.(map[string]any); ok {
						url, _ := rm["url"].(string)
						title, _ := rm["title"].(string)
						pageAge := rm["page_age"]
						content = append(content, languagemodel.SourceURL{
							ID:    m.generateID(),
							URL:   url,
							Title: strPtr(title),
							ProviderMetadata: shared.ProviderMetadata{
								"anthropic": jsonvalue.JSONObject{
									"pageAge": pageAge,
								},
							},
						})
					}
				}
			} else if contentMap, ok := contentRaw.(map[string]any); ok {
				isErr := true
				errorCode, _ := contentMap["error_code"].(string)
				content = append(content, languagemodel.ToolResult{
					ToolCallID: toolUseID,
					ToolName:   prepared.toolNameMapping.ToCustomToolName("web_search"),
					IsError:    &isErr,
					Result: map[string]any{
						"type":      "web_search_tool_result_error",
						"errorCode": errorCode,
					},
				})
			}

		case "code_execution_tool_result":
			toolUseID, _ := part["tool_use_id"].(string)
			contentMap, _ := part["content"].(map[string]any)
			processCodeExecutionToolResult(contentMap, toolUseID, prepared.toolNameMapping, &content)

		case "bash_code_execution_tool_result", "text_editor_code_execution_tool_result":
			toolUseID, _ := part["tool_use_id"].(string)
			contentMap := part["content"]
			content = append(content, languagemodel.ToolResult{
				ToolCallID: toolUseID,
				ToolName:   prepared.toolNameMapping.ToCustomToolName("code_execution"),
				Result:     contentMap,
			})

		case "tool_search_tool_result":
			toolUseID, _ := part["tool_use_id"].(string)
			contentMap, _ := part["content"].(map[string]any)
			processToolSearchResult(contentMap, toolUseID, serverToolCalls, prepared.toolNameMapping, &content)
		}
	}

	// Build raw usage
	rawUsage := usageToMap(response.Usage)

	finishReason := languagemodel.FinishReason{
		Unified: mapAnthropicStopReason(response.StopReason, isJsonResponseFromTool),
		Raw:     response.StopReason,
	}

	usage := convertAnthropicMessagesUsage(response.Usage, rawUsage)

	// Provider metadata
	anthropicMetadata := buildAnthropicMetadata(rawUsage, response)

	providerMetadata := shared.ProviderMetadata{
		"anthropic": anthropicMetadata,
	}
	if prepared.usedCustomProviderKey && prepared.providerOptionsName != "anthropic" {
		providerMetadata[prepared.providerOptionsName] = anthropicMetadata
	}

	return languagemodel.GenerateResult{
		Content:      content,
		FinishReason: finishReason,
		Usage:        usage,
		ProviderMetadata: providerMetadata,
		Request:  &languagemodel.GenerateResultRequest{Body: prepared.args},
		Response: &languagemodel.GenerateResultResponse{
			ResponseMetadata: languagemodel.ResponseMetadata{
				ID:      response.ID,
				ModelID: response.Model,
			},
			Headers: result.ResponseHeaders,
			Body:    result.RawValue,
		},
		Warnings: prepared.warnings,
	}, nil
}

// DoStream performs a streaming request to the Anthropic Messages API.
func (m *AnthropicMessagesLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	userBetas := m.getBetasFromHeaders(options.Headers)

	prepared, err := m.getArgs(options, true, userBetas)
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	citationDocuments := m.extractCitationDocuments(options.Prompt)
	markCodeExecutionDynamic := hasWebTool20260209WithoutCodeExecution(prepared.args)

	ctx := options.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	url := m.buildRequestURL(true)
	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[<-chan providerutils.ParseResult[AnthropicMessagesChunk]]{
		URL:     url,
		Headers: m.getHeaders(prepared.betas, options.Headers),
		Body:    m.transformRequestBody(prepared.args, prepared.betas),
		FailedResponseHandler:     wrapErrorResponseHandler(anthropicFailedResponseHandler),
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(anthropicMessagesChunkSchema),
		Ctx:   ctx,
		Fetch: m.config.Fetch,
	})
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	eventStream := result.Value
	streamCh := make(chan languagemodel.StreamPart, 64)

	go func() {
		defer close(streamCh)

		// Send stream-start
		streamCh <- languagemodel.StreamPartStreamStart{Warnings: prepared.warnings}

		finishReason := languagemodel.FinishReason{
			Unified: languagemodel.FinishReasonOther,
		}
		usage := AnthropicMessagesUsage{}
		contentBlocks := map[int]*streamContentBlock{}
		mcpToolCalls := map[string]languagemodel.ToolCall{}
		serverToolCalls := map[string]string{}
		var rawUsage jsonvalue.JSONObject
		var cacheCreationInputTokens *int
		var stopSequence *string
		var container *AnthropicContainerInfo
		var contextManagementResp *AnthropicContextManagementResponse
		isJsonResponseFromTool := false
		var blockType string

		includeRawChunks := false
		if options.IncludeRawChunks != nil {
			includeRawChunks = *options.IncludeRawChunks
		}

		for chunk := range eventStream {
			if includeRawChunks {
				streamCh <- languagemodel.StreamPartRaw{RawValue: chunk.RawValue}
			}

			if !chunk.Success {
				streamCh <- languagemodel.StreamPartError{Error: chunk.Error}
				continue
			}

			value := chunk.Value

			switch value.Type {
			case "ping":
				// ignored

			case "content_block_start":
				contentBlock := value.ContentBlock
				contentBlockType, _ := contentBlock["type"].(string)
				blockType = contentBlockType
				index := 0
				if value.Index != nil {
					index = *value.Index
				}

				switch contentBlockType {
				case "text":
					if prepared.usesJsonResponseTool {
						continue
					}
					contentBlocks[index] = &streamContentBlock{blockType: "text"}
					streamCh <- languagemodel.StreamPartTextStart{ID: fmt.Sprintf("%d", index)}

				case "thinking":
					contentBlocks[index] = &streamContentBlock{blockType: "reasoning"}
					streamCh <- languagemodel.StreamPartReasoningStart{ID: fmt.Sprintf("%d", index)}

				case "redacted_thinking":
					contentBlocks[index] = &streamContentBlock{blockType: "reasoning"}
					data, _ := contentBlock["data"].(string)
					streamCh <- languagemodel.StreamPartReasoningStart{
						ID: fmt.Sprintf("%d", index),
						ProviderMetadata: shared.ProviderMetadata{
							"anthropic": jsonvalue.JSONObject{
								"redactedData": data,
							},
						},
					}

				case "compaction":
					contentBlocks[index] = &streamContentBlock{blockType: "text"}
					streamCh <- languagemodel.StreamPartTextStart{
						ID: fmt.Sprintf("%d", index),
						ProviderMetadata: shared.ProviderMetadata{
							"anthropic": jsonvalue.JSONObject{
								"type": "compaction",
							},
						},
					}

				case "tool_use":
					partName, _ := contentBlock["name"].(string)
					partID, _ := contentBlock["id"].(string)

					if prepared.usesJsonResponseTool && partName == "json" {
						isJsonResponseFromTool = true
						contentBlocks[index] = &streamContentBlock{blockType: "text"}
						streamCh <- languagemodel.StreamPartTextStart{ID: fmt.Sprintf("%d", index)}
					} else {
						// Check for non-empty input (programmatic tool calling)
						initialInput := ""
						if inputRaw, ok := contentBlock["input"]; ok && inputRaw != nil {
							if m2, ok := inputRaw.(map[string]any); ok && len(m2) > 0 {
								inputJSON, _ := json.Marshal(m2)
								initialInput = string(inputJSON)
							}
						}

						var callerInfo map[string]any
						if callerRaw, ok := contentBlock["caller"]; ok && callerRaw != nil {
							if cm, ok := callerRaw.(map[string]any); ok {
								ct, _ := cm["type"].(string)
								callerInfo = map[string]any{"type": ct}
								if tid, ok := cm["tool_id"].(string); ok {
									callerInfo["toolId"] = tid
								}
							}
						}

						contentBlocks[index] = &streamContentBlock{
							blockType:  "tool-call",
							toolCallID: partID,
							toolName:   partName,
							input:      initialInput,
							firstDelta: len(initialInput) == 0,
							callerInfo: callerInfo,
						}
						streamCh <- languagemodel.StreamPartToolInputStart{
							ID:       partID,
							ToolName: partName,
						}
					}

				case "server_tool_use":
					partName, _ := contentBlock["name"].(string)
					partID, _ := contentBlock["id"].(string)

					if partName == "web_fetch" || partName == "web_search" ||
						partName == "code_execution" ||
						partName == "text_editor_code_execution" ||
						partName == "bash_code_execution" {
						providerToolName := partName
						if partName == "text_editor_code_execution" || partName == "bash_code_execution" {
							providerToolName = "code_execution"
						}
						customToolName := prepared.toolNameMapping.ToCustomToolName(providerToolName)

						finalInput := ""
						if inputRaw, ok := contentBlock["input"]; ok && inputRaw != nil {
							if m2, ok := inputRaw.(map[string]any); ok && len(m2) > 0 {
								inputJSON, _ := json.Marshal(m2)
								finalInput = string(inputJSON)
							}
						}

						pe := true
						cb := &streamContentBlock{
							blockType:        "tool-call",
							toolCallID:       partID,
							toolName:         customToolName,
							input:            finalInput,
							providerExecuted: &pe,
							firstDelta:       true,
							providerToolName: partName,
						}
						if markCodeExecutionDynamic && providerToolName == "code_execution" {
							d := true
							cb.dynamic = &d
						}
						contentBlocks[index] = cb

						sp := languagemodel.StreamPartToolInputStart{
							ID:               partID,
							ToolName:         customToolName,
							ProviderExecuted: &pe,
						}
						if markCodeExecutionDynamic && providerToolName == "code_execution" {
							d := true
							sp.Dynamic = &d
						}
						streamCh <- sp
					} else if partName == "tool_search_tool_regex" || partName == "tool_search_tool_bm25" {
						serverToolCalls[partID] = partName
						customToolName := prepared.toolNameMapping.ToCustomToolName(partName)
						pe := true
						contentBlocks[index] = &streamContentBlock{
							blockType:        "tool-call",
							toolCallID:       partID,
							toolName:         customToolName,
							input:            "",
							providerExecuted: &pe,
							firstDelta:       true,
							providerToolName: partName,
						}
						streamCh <- languagemodel.StreamPartToolInputStart{
							ID:               partID,
							ToolName:         customToolName,
							ProviderExecuted: &pe,
						}
					}

				case "web_fetch_tool_result":
					toolUseID, _ := contentBlock["tool_use_id"].(string)
					contentData, _ := contentBlock["content"].(map[string]any)
					contentType2, _ := contentData["type"].(string)

					if contentType2 == "web_fetch_result" {
						innerContent, _ := contentData["content"].(map[string]any)
						fetchTitle := ""
						if t, ok := innerContent["title"].(string); ok {
							fetchTitle = t
						} else if u, ok := contentData["url"].(string); ok {
							fetchTitle = u
						}
						source, _ := innerContent["source"].(map[string]any)
						mediaType, _ := source["media_type"].(string)
						citationDocuments = append(citationDocuments, citationDocumentInfo{
							Title:     fetchTitle,
							MediaType: mediaType,
						})
						streamCh <- languagemodel.ToolResult{
							ToolCallID: toolUseID,
							ToolName:   prepared.toolNameMapping.ToCustomToolName("web_fetch"),
							Result:     buildWebFetchResult(contentData),
						}
					} else if contentType2 == "web_fetch_tool_result_error" {
						isErr := true
						errorCode, _ := contentData["error_code"].(string)
						streamCh <- languagemodel.ToolResult{
							ToolCallID: toolUseID,
							ToolName:   prepared.toolNameMapping.ToCustomToolName("web_fetch"),
							IsError:    &isErr,
							Result: map[string]any{
								"type":      "web_fetch_tool_result_error",
								"errorCode": errorCode,
							},
						}
					}

				case "web_search_tool_result":
					toolUseID, _ := contentBlock["tool_use_id"].(string)
					contentRaw := contentBlock["content"]
					processWebSearchStreamResult(contentRaw, toolUseID, prepared.toolNameMapping, m.generateID, streamCh)

				case "code_execution_tool_result":
					toolUseID, _ := contentBlock["tool_use_id"].(string)
					contentData, _ := contentBlock["content"].(map[string]any)
					processCodeExecutionStreamResult(contentData, toolUseID, prepared.toolNameMapping, streamCh)

				case "bash_code_execution_tool_result", "text_editor_code_execution_tool_result":
					toolUseID, _ := contentBlock["tool_use_id"].(string)
					contentData := contentBlock["content"]
					streamCh <- languagemodel.ToolResult{
						ToolCallID: toolUseID,
						ToolName:   prepared.toolNameMapping.ToCustomToolName("code_execution"),
						Result:     contentData,
					}

				case "tool_search_tool_result":
					toolUseID, _ := contentBlock["tool_use_id"].(string)
					contentData, _ := contentBlock["content"].(map[string]any)
					processToolSearchStreamResult(contentData, toolUseID, serverToolCalls, prepared.toolNameMapping, streamCh)

				case "mcp_tool_use":
					partID, _ := contentBlock["id"].(string)
					partName, _ := contentBlock["name"].(string)
					serverName, _ := contentBlock["server_name"].(string)
					input := contentBlock["input"]
					inputJSON, _ := json.Marshal(input)
					pe := true
					d := true
					tc := languagemodel.ToolCall{
						ToolCallID:       partID,
						ToolName:         partName,
						Input:            string(inputJSON),
						ProviderExecuted: &pe,
						Dynamic:          &d,
						ProviderMetadata: shared.ProviderMetadata{
							"anthropic": jsonvalue.JSONObject{
								"type":       "mcp-tool-use",
								"serverName": serverName,
							},
						},
					}
					mcpToolCalls[partID] = tc
					streamCh <- tc

				case "mcp_tool_result":
					toolUseID, _ := contentBlock["tool_use_id"].(string)
					isError, _ := contentBlock["is_error"].(bool)
					resultContent := contentBlock["content"]
					mcpTC := mcpToolCalls[toolUseID]
					d := true
					streamCh <- languagemodel.ToolResult{
						ToolCallID:       toolUseID,
						ToolName:         mcpTC.ToolName,
						IsError:          &isError,
						Result:           resultContent,
						Dynamic:          &d,
						ProviderMetadata: mcpTC.ProviderMetadata,
					}
				}

			case "content_block_stop":
				index := 0
				if value.Index != nil {
					index = *value.Index
				}
				if cb, ok := contentBlocks[index]; ok {
					switch cb.blockType {
					case "text":
						streamCh <- languagemodel.StreamPartTextEnd{ID: fmt.Sprintf("%d", index)}
					case "reasoning":
						streamCh <- languagemodel.StreamPartReasoningEnd{ID: fmt.Sprintf("%d", index)}
					case "tool-call":
						isJsonRespTool := prepared.usesJsonResponseTool && cb.toolName == "json"
						if !isJsonRespTool {
							streamCh <- languagemodel.StreamPartToolInputEnd{ID: cb.toolCallID}

							finalInput := cb.input
							if finalInput == "" {
								finalInput = "{}"
							}
							// For code_execution, inject 'programmatic-tool-call' type
							if cb.providerToolName == "code_execution" {
								var parsed map[string]any
								if err := json.Unmarshal([]byte(finalInput), &parsed); err == nil {
									if _, hasCode := parsed["code"]; hasCode {
										if _, hasType := parsed["type"]; !hasType {
											parsed["type"] = "programmatic-tool-call"
											reencoded, _ := json.Marshal(parsed)
											finalInput = string(reencoded)
										}
									}
								}
							}

							tc := languagemodel.ToolCall{
								ToolCallID:       cb.toolCallID,
								ToolName:         cb.toolName,
								Input:            finalInput,
								ProviderExecuted: cb.providerExecuted,
							}
							if markCodeExecutionDynamic && cb.providerToolName == "code_execution" {
								d := true
								tc.Dynamic = &d
							}
							if cb.callerInfo != nil {
								tc.ProviderMetadata = shared.ProviderMetadata{
									"anthropic": jsonvalue.JSONObject{
										"caller": cb.callerInfo,
									},
								}
							}
							streamCh <- tc
						}
					}
					delete(contentBlocks, index)
				}
				blockType = ""

			case "content_block_delta":
				delta := value.Delta
				deltaType, _ := delta["type"].(string)
				index := 0
				if value.Index != nil {
					index = *value.Index
				}

				switch deltaType {
				case "text_delta":
					if prepared.usesJsonResponseTool {
						continue
					}
					text, _ := delta["text"].(string)
					streamCh <- languagemodel.StreamPartTextDelta{
						ID:    fmt.Sprintf("%d", index),
						Delta: text,
					}

				case "thinking_delta":
					thinking, _ := delta["thinking"].(string)
					streamCh <- languagemodel.StreamPartReasoningDelta{
						ID:    fmt.Sprintf("%d", index),
						Delta: thinking,
					}

				case "signature_delta":
					if blockType == "thinking" {
						sig, _ := delta["signature"].(string)
						streamCh <- languagemodel.StreamPartReasoningDelta{
							ID:    fmt.Sprintf("%d", index),
							Delta: "",
							ProviderMetadata: shared.ProviderMetadata{
								"anthropic": jsonvalue.JSONObject{
									"signature": sig,
								},
							},
						}
					}

				case "compaction_delta":
					if content, ok := delta["content"].(string); ok {
						streamCh <- languagemodel.StreamPartTextDelta{
							ID:    fmt.Sprintf("%d", index),
							Delta: content,
						}
					}

				case "input_json_delta":
					cb := contentBlocks[index]
					partialJSON, _ := delta["partial_json"].(string)

					if len(partialJSON) == 0 {
						continue
					}

					if isJsonResponseFromTool {
						if cb == nil || cb.blockType != "text" {
							continue
						}
						streamCh <- languagemodel.StreamPartTextDelta{
							ID:    fmt.Sprintf("%d", index),
							Delta: partialJSON,
						}
					} else {
						if cb == nil || cb.blockType != "tool-call" {
							continue
						}
						d := partialJSON
						if cb.firstDelta &&
							(cb.providerToolName == "bash_code_execution" ||
								cb.providerToolName == "text_editor_code_execution") {
							d = fmt.Sprintf(`{"type": "%s",%s`, cb.providerToolName, d[1:])
						}
						streamCh <- languagemodel.StreamPartToolInputDelta{
							ID:    cb.toolCallID,
							Delta: d,
						}
						cb.input += d
						cb.firstDelta = false
					}

				case "citations_delta":
					citationRaw, _ := delta["citation"].(map[string]any)
					source := createCitationSource(citationRaw, citationDocuments, m.generateID)
					if source != nil {
						streamCh <- source
					}
				}

			case "message_start":
				if value.Message != nil {
					msg := value.Message
					usage.InputTokens = msg.Usage.InputTokens
					if msg.Usage.CacheReadInputTokens != nil {
						usage.CacheReadInputTokens = msg.Usage.CacheReadInputTokens
					}
					if msg.Usage.CacheCreationInputTokens != nil {
						usage.CacheCreationInputTokens = msg.Usage.CacheCreationInputTokens
						cacheCreationInputTokens = msg.Usage.CacheCreationInputTokens
					}

					rawUsage = usageToMap(msg.Usage)

					if msg.Container != nil {
						container = &AnthropicContainerInfo{
							ExpiresAt: msg.Container.ExpiresAt,
							ID:        msg.Container.ID,
						}
					}

					if msg.StopReason != nil {
						finishReason = languagemodel.FinishReason{
							Unified: mapAnthropicStopReason(msg.StopReason, isJsonResponseFromTool),
							Raw:     msg.StopReason,
						}
					}

					streamCh <- languagemodel.StreamPartResponseMetadata{
						ResponseMetadata: languagemodel.ResponseMetadata{
							ID:      msg.ID,
							ModelID: msg.Model,
						},
					}

					// Process pre-populated content blocks (programmatic tool calling)
					if msg.Content != nil {
						for _, part := range msg.Content {
							partType, _ := part["type"].(string)
							if partType == "tool_use" {
								partID, _ := part["id"].(string)
								partName, _ := part["name"].(string)
								input := part["input"]
								inputStr := "{}"
								if input != nil {
									inputJSON, _ := json.Marshal(input)
									inputStr = string(inputJSON)
								}

								streamCh <- languagemodel.StreamPartToolInputStart{
									ID:       partID,
									ToolName: partName,
								}
								streamCh <- languagemodel.StreamPartToolInputDelta{
									ID:    partID,
									Delta: inputStr,
								}
								streamCh <- languagemodel.StreamPartToolInputEnd{ID: partID}

								tc := languagemodel.ToolCall{
									ToolCallID: partID,
									ToolName:   partName,
									Input:      inputStr,
								}
								if callerRaw, ok := part["caller"]; ok && callerRaw != nil {
									if cm, ok := callerRaw.(map[string]any); ok {
										ct, _ := cm["type"].(string)
										callerInfo := map[string]any{"type": ct}
										if tid, ok := cm["tool_id"].(string); ok {
											callerInfo["toolId"] = tid
										}
										tc.ProviderMetadata = shared.ProviderMetadata{
											"anthropic": jsonvalue.JSONObject{
												"caller": callerInfo,
											},
										}
									}
								}
								streamCh <- tc
							}
						}
					}
				}

			case "message_delta":
				if value.Usage != nil {
					if value.Usage.InputTokens != 0 && usage.InputTokens != value.Usage.InputTokens {
						usage.InputTokens = value.Usage.InputTokens
					}
					usage.OutputTokens = value.Usage.OutputTokens
					if value.Usage.CacheReadInputTokens != nil {
						usage.CacheReadInputTokens = value.Usage.CacheReadInputTokens
					}
					if value.Usage.CacheCreationInputTokens != nil {
						usage.CacheCreationInputTokens = value.Usage.CacheCreationInputTokens
						cacheCreationInputTokens = value.Usage.CacheCreationInputTokens
					}
					if value.Usage.Iterations != nil {
						usage.Iterations = value.Usage.Iterations
					}
				}

				delta := value.Delta
				if delta != nil {
					sr, _ := delta["stop_reason"].(string)
					if sr != "" {
						finishReason = languagemodel.FinishReason{
							Unified: mapAnthropicStopReason(&sr, isJsonResponseFromTool),
							Raw:     &sr,
						}
					}
					if ss, ok := delta["stop_sequence"].(string); ok {
						stopSequence = &ss
					}
					if containerRaw, ok := delta["container"]; ok && containerRaw != nil {
						if cm, ok := containerRaw.(map[string]any); ok {
							expiresAt, _ := cm["expires_at"].(string)
							containerID, _ := cm["id"].(string)
							container = &AnthropicContainerInfo{
								ExpiresAt: expiresAt,
								ID:        containerID,
							}
							if skillsRaw, ok := cm["skills"].([]any); ok {
								for _, sr := range skillsRaw {
									if sm, ok := sr.(map[string]any); ok {
										st, _ := sm["type"].(string)
										sid, _ := sm["skill_id"].(string)
										sv, _ := sm["version"].(string)
										container.Skills = append(container.Skills, AnthropicContainerSkill{
											Type:    st,
											SkillID: sid,
											Version: sv,
										})
									}
								}
							}
						}
					}
					if ctxMgmt, ok := delta["context_management"]; ok && ctxMgmt != nil {
						if cm, ok := ctxMgmt.(map[string]any); ok {
							contextManagementResp = mapResponseContextManagement(cm)
						}
					}
				}

				// Merge rawUsage
				if value.Usage != nil {
					newRawUsage := usageToMap(*value.Usage)
					if rawUsage == nil {
						rawUsage = newRawUsage
					} else {
						for k, v := range newRawUsage {
							rawUsage[k] = v
						}
					}
				}

			case "message_stop":
				anthropicMetadata := jsonvalue.JSONObject{
					"usage":                    rawUsage,
					"cacheCreationInputTokens": anyFromIntPtr(cacheCreationInputTokens),
					"stopSequence":             anyFromStrPtr(stopSequence),
				}
				if usage.Iterations != nil {
					iters := make([]any, 0, len(usage.Iterations))
					for _, iter := range usage.Iterations {
						iters = append(iters, map[string]any{
							"type":         iter.Type,
							"inputTokens":  iter.InputTokens,
							"outputTokens": iter.OutputTokens,
						})
					}
					anthropicMetadata["iterations"] = iters
				} else {
					anthropicMetadata["iterations"] = nil
				}
				if container != nil {
					anthropicMetadata["container"] = containerInfoToMap(container)
				} else {
					anthropicMetadata["container"] = nil
				}
				if contextManagementResp != nil {
					anthropicMetadata["contextManagement"] = contextManagementRespToMap(contextManagementResp)
				} else {
					anthropicMetadata["contextManagement"] = nil
				}

				providerMetadata := shared.ProviderMetadata{
					"anthropic": anthropicMetadata,
				}
				if prepared.usedCustomProviderKey && prepared.providerOptionsName != "anthropic" {
					providerMetadata[prepared.providerOptionsName] = anthropicMetadata
				}

				streamCh <- languagemodel.StreamPartFinish{
					FinishReason:     finishReason,
					Usage:            convertAnthropicMessagesUsage(usage, rawUsage),
					ProviderMetadata: providerMetadata,
				}

			case "error":
				if errData, ok := value.Delta["message"].(string); ok {
					streamCh <- languagemodel.StreamPartError{Error: fmt.Errorf("%s", errData)}
				} else {
					streamCh <- languagemodel.StreamPartError{Error: fmt.Errorf("unknown stream error")}
				}
			}
		}
	}()

	return languagemodel.StreamResult{
		Stream:   streamCh,
		Request:  &languagemodel.StreamResultRequest{Body: prepared.args},
		Response: &languagemodel.StreamResultResponse{Headers: result.ResponseHeaders},
	}, nil
}

// --- Helper types and functions ---

type streamContentBlock struct {
	blockType        string // "text", "reasoning", "tool-call"
	toolCallID       string
	toolName         string
	input            string
	providerExecuted *bool
	dynamic          *bool
	firstDelta       bool
	providerToolName string
	callerInfo       map[string]any
}

// getModelCapabilities returns capabilities for known Anthropic models.
type modelCapabilities struct {
	maxOutputTokens          int
	supportsStructuredOutput bool
	isKnownModel             bool
}

func getModelCapabilities(modelID string) modelCapabilities {
	if strings.Contains(modelID, "claude-sonnet-4-6") || strings.Contains(modelID, "claude-opus-4-6") {
		return modelCapabilities{128000, true, true}
	}
	if strings.Contains(modelID, "claude-sonnet-4-5") || strings.Contains(modelID, "claude-opus-4-5") || strings.Contains(modelID, "claude-haiku-4-5") {
		return modelCapabilities{64000, true, true}
	}
	if strings.Contains(modelID, "claude-opus-4-1") {
		return modelCapabilities{32000, true, true}
	}
	if strings.Contains(modelID, "claude-sonnet-4-") {
		return modelCapabilities{64000, false, true}
	}
	if strings.Contains(modelID, "claude-opus-4-") {
		return modelCapabilities{32000, false, true}
	}
	if strings.Contains(modelID, "claude-3-haiku") {
		return modelCapabilities{4096, false, true}
	}
	return modelCapabilities{4096, false, false}
}

// hasWebTool20260209WithoutCodeExecution checks if web tools 20260209 are present without code execution.
func hasWebTool20260209WithoutCodeExecution(args map[string]any) bool {
	toolsRaw, ok := args["tools"]
	if !ok || toolsRaw == nil {
		return false
	}
	tools, ok := toolsRaw.([]AnthropicTool)
	if !ok {
		return false
	}

	hasWeb := false
	hasCodeExec := false
	for _, tool := range tools {
		if tool.Type == "web_fetch_20260209" || tool.Type == "web_search_20260209" {
			hasWeb = true
			continue
		}
		if tool.Name == "code_execution" {
			hasCodeExec = true
			break
		}
	}
	return hasWeb && !hasCodeExec
}

// createCitationSource creates a citation source from a citation map.
func createCitationSource(citation map[string]any, docs []citationDocumentInfo, genID func() string) languagemodel.Source {
	citType, _ := citation["type"].(string)

	if citType == "web_search_result_location" {
		url, _ := citation["url"].(string)
		title, _ := citation["title"].(string)
		citedText, _ := citation["cited_text"].(string)
		encryptedIndex, _ := citation["encrypted_index"].(string)
		return languagemodel.SourceURL{
			ID:    genID(),
			URL:   url,
			Title: strPtr(title),
			ProviderMetadata: shared.ProviderMetadata{
				"anthropic": jsonvalue.JSONObject{
					"citedText":      citedText,
					"encryptedIndex": encryptedIndex,
				},
			},
		}
	}

	if citType != "page_location" && citType != "char_location" {
		return nil
	}

	docIndex := 0
	if di, ok := citation["document_index"].(float64); ok {
		docIndex = int(di)
	}
	if docIndex >= len(docs) {
		return nil
	}

	docInfo := docs[docIndex]
	title := docInfo.Title
	if dt, ok := citation["document_title"].(string); ok && dt != "" {
		title = dt
	}

	citedText, _ := citation["cited_text"].(string)
	metadata := jsonvalue.JSONObject{
		"citedText": citedText,
	}

	if citType == "page_location" {
		if sp, ok := citation["start_page_number"].(float64); ok {
			metadata["startPageNumber"] = int(sp)
		}
		if ep, ok := citation["end_page_number"].(float64); ok {
			metadata["endPageNumber"] = int(ep)
		}
	} else {
		if sc, ok := citation["start_char_index"].(float64); ok {
			metadata["startCharIndex"] = int(sc)
		}
		if ec, ok := citation["end_char_index"].(float64); ok {
			metadata["endCharIndex"] = int(ec)
		}
	}

	return languagemodel.SourceDocument{
		ID:        genID(),
		MediaType: docInfo.MediaType,
		Title:     title,
		Filename:  docInfo.Filename,
		ProviderMetadata: shared.ProviderMetadata{
			"anthropic": metadata,
		},
	}
}

// mapResponseContextManagement maps API context management response to metadata format.
func mapResponseContextManagement(cm map[string]any) *AnthropicContextManagementResponse {
	appliedEditsRaw, ok := cm["applied_edits"].([]any)
	if !ok {
		return nil
	}
	appliedEdits := make([]map[string]any, 0, len(appliedEditsRaw))
	for _, editRaw := range appliedEditsRaw {
		editMap, ok := editRaw.(map[string]any)
		if !ok {
			continue
		}
		editType, _ := editMap["type"].(string)
		switch editType {
		case "clear_tool_uses_20250919":
			clearedToolUses, _ := editMap["cleared_tool_uses"].(float64)
			clearedInputTokens, _ := editMap["cleared_input_tokens"].(float64)
			appliedEdits = append(appliedEdits, map[string]any{
				"type":               editType,
				"clearedToolUses":    int(clearedToolUses),
				"clearedInputTokens": int(clearedInputTokens),
			})
		case "clear_thinking_20251015":
			clearedThinkingTurns, _ := editMap["cleared_thinking_turns"].(float64)
			clearedInputTokens, _ := editMap["cleared_input_tokens"].(float64)
			appliedEdits = append(appliedEdits, map[string]any{
				"type":                 editType,
				"clearedThinkingTurns": int(clearedThinkingTurns),
				"clearedInputTokens":   int(clearedInputTokens),
			})
		case "compact_20260112":
			appliedEdits = append(appliedEdits, map[string]any{
				"type": editType,
			})
		}
	}
	return &AnthropicContextManagementResponse{
		AppliedEdits: appliedEdits,
	}
}

// buildAnthropicMetadata builds the provider metadata from the response.
func buildAnthropicMetadata(rawUsage jsonvalue.JSONObject, response AnthropicMessagesResponse) jsonvalue.JSONObject {
	metadata := jsonvalue.JSONObject{
		"usage":                    rawUsage,
		"cacheCreationInputTokens": anyFromIntPtr(response.Usage.CacheCreationInputTokens),
		"stopSequence":             anyFromStrPtr(response.StopSequence),
	}

	if response.Usage.Iterations != nil {
		iters := make([]any, 0, len(response.Usage.Iterations))
		for _, iter := range response.Usage.Iterations {
			iters = append(iters, map[string]any{
				"type":         iter.Type,
				"inputTokens":  iter.InputTokens,
				"outputTokens": iter.OutputTokens,
			})
		}
		metadata["iterations"] = iters
	} else {
		metadata["iterations"] = nil
	}

	if response.Container != nil {
		containerMap := map[string]any{
			"expiresAt": response.Container.ExpiresAt,
			"id":        response.Container.ID,
		}
		if response.Container.Skills != nil {
			skills := make([]any, 0, len(response.Container.Skills))
			for _, sk := range response.Container.Skills {
				skills = append(skills, map[string]any{
					"type":    sk.Type,
					"skillId": sk.SkillID,
					"version": sk.Version,
				})
			}
			containerMap["skills"] = skills
		} else {
			containerMap["skills"] = nil
		}
		metadata["container"] = containerMap
	} else {
		metadata["container"] = nil
	}

	if response.ContextManagement != nil {
		metadata["contextManagement"] = mapResponseContextManagementFromAPI(response.ContextManagement)
	} else {
		metadata["contextManagement"] = nil
	}

	return metadata
}

func mapResponseContextManagementFromAPI(cm *struct {
	AppliedEdits []map[string]any `json:"applied_edits"`
}) map[string]any {
	if cm == nil {
		return nil
	}
	result := map[string]any{}
	edits := make([]map[string]any, 0, len(cm.AppliedEdits))
	for _, edit := range cm.AppliedEdits {
		editType, _ := edit["type"].(string)
		switch editType {
		case "clear_tool_uses_20250919":
			clearedToolUses, _ := edit["cleared_tool_uses"].(float64)
			clearedInputTokens, _ := edit["cleared_input_tokens"].(float64)
			edits = append(edits, map[string]any{
				"type":               editType,
				"clearedToolUses":    int(clearedToolUses),
				"clearedInputTokens": int(clearedInputTokens),
			})
		case "clear_thinking_20251015":
			clearedThinkingTurns, _ := edit["cleared_thinking_turns"].(float64)
			clearedInputTokens, _ := edit["cleared_input_tokens"].(float64)
			edits = append(edits, map[string]any{
				"type":                 editType,
				"clearedThinkingTurns": int(clearedThinkingTurns),
				"clearedInputTokens":   int(clearedInputTokens),
			})
		case "compact_20260112":
			edits = append(edits, map[string]any{
				"type": editType,
			})
		}
	}
	result["appliedEdits"] = edits
	return result
}

// processCodeExecutionToolResult processes code_execution_tool_result content blocks.
func processCodeExecutionToolResult(
	contentMap map[string]any,
	toolUseID string,
	toolNameMapping providerutils.ToolNameMapping,
	content *[]languagemodel.Content,
) {
	contentType, _ := contentMap["type"].(string)
	switch contentType {
	case "code_execution_result":
		*content = append(*content, languagemodel.ToolResult{
			ToolCallID: toolUseID,
			ToolName:   toolNameMapping.ToCustomToolName("code_execution"),
			Result: map[string]any{
				"type":        contentType,
				"stdout":      contentMap["stdout"],
				"stderr":      contentMap["stderr"],
				"return_code": contentMap["return_code"],
				"content":     defaultSlice(contentMap["content"]),
			},
		})
	case "encrypted_code_execution_result":
		*content = append(*content, languagemodel.ToolResult{
			ToolCallID: toolUseID,
			ToolName:   toolNameMapping.ToCustomToolName("code_execution"),
			Result: map[string]any{
				"type":             contentType,
				"encrypted_stdout": contentMap["encrypted_stdout"],
				"stderr":           contentMap["stderr"],
				"return_code":      contentMap["return_code"],
				"content":          defaultSlice(contentMap["content"]),
			},
		})
	case "code_execution_tool_result_error":
		isErr := true
		*content = append(*content, languagemodel.ToolResult{
			ToolCallID: toolUseID,
			ToolName:   toolNameMapping.ToCustomToolName("code_execution"),
			IsError:    &isErr,
			Result: map[string]any{
				"type":      "code_execution_tool_result_error",
				"errorCode": contentMap["error_code"],
			},
		})
	}
}

// processToolSearchResult processes tool_search_tool_result content blocks.
func processToolSearchResult(
	contentMap map[string]any,
	toolUseID string,
	serverToolCallsMap map[string]string,
	toolNameMapping providerutils.ToolNameMapping,
	content *[]languagemodel.Content,
) {
	providerToolName := serverToolCallsMap[toolUseID]
	if providerToolName == "" {
		bm25Custom := toolNameMapping.ToCustomToolName("tool_search_tool_bm25")
		regexCustom := toolNameMapping.ToCustomToolName("tool_search_tool_regex")
		if bm25Custom != "tool_search_tool_bm25" {
			providerToolName = "tool_search_tool_bm25"
		} else if regexCustom != "tool_search_tool_regex" {
			providerToolName = "tool_search_tool_regex"
		} else {
			providerToolName = "tool_search_tool_regex"
		}
	}

	contentType, _ := contentMap["type"].(string)
	if contentType == "tool_search_tool_search_result" {
		refs, _ := contentMap["tool_references"].([]any)
		results := make([]map[string]any, 0, len(refs))
		for _, r := range refs {
			if rm, ok := r.(map[string]any); ok {
				results = append(results, map[string]any{
					"type":     rm["type"],
					"toolName": rm["tool_name"],
				})
			}
		}
		*content = append(*content, languagemodel.ToolResult{
			ToolCallID: toolUseID,
			ToolName:   toolNameMapping.ToCustomToolName(providerToolName),
			Result:     results,
		})
	} else {
		isErr := true
		*content = append(*content, languagemodel.ToolResult{
			ToolCallID: toolUseID,
			ToolName:   toolNameMapping.ToCustomToolName(providerToolName),
			IsError:    &isErr,
			Result: map[string]any{
				"type":      "tool_search_tool_result_error",
				"errorCode": contentMap["error_code"],
			},
		})
	}
}

// buildWebFetchResult builds the web fetch result map.
func buildWebFetchResult(contentData map[string]any) map[string]any {
	innerContent, _ := contentData["content"].(map[string]any)
	innerSource, _ := innerContent["source"].(map[string]any)
	return map[string]any{
		"type":        "web_fetch_result",
		"url":         contentData["url"],
		"retrievedAt": contentData["retrieved_at"],
		"content": map[string]any{
			"type":      innerContent["type"],
			"title":     innerContent["title"],
			"citations": innerContent["citations"],
			"source": map[string]any{
				"type":      innerSource["type"],
				"mediaType": innerSource["media_type"],
				"data":      innerSource["data"],
			},
		},
	}
}

// processWebSearchStreamResult processes web search results in streaming.
func processWebSearchStreamResult(
	contentRaw any,
	toolUseID string,
	toolNameMapping providerutils.ToolNameMapping,
	genID func() string,
	ch chan<- languagemodel.StreamPart,
) {
	if contentArr, ok := contentRaw.([]any); ok {
		results := make([]map[string]any, 0, len(contentArr))
		for _, r := range contentArr {
			if rm, ok := r.(map[string]any); ok {
				result := map[string]any{
					"url":              rm["url"],
					"title":            rm["title"],
					"type":             rm["type"],
					"encryptedContent": rm["encrypted_content"],
				}
				if pa, ok := rm["page_age"]; ok {
					result["pageAge"] = pa
				} else {
					result["pageAge"] = nil
				}
				results = append(results, result)
			}
		}
		ch <- languagemodel.ToolResult{
			ToolCallID: toolUseID,
			ToolName:   toolNameMapping.ToCustomToolName("web_search"),
			Result:     results,
		}
		for _, r := range contentArr {
			if rm, ok := r.(map[string]any); ok {
				url, _ := rm["url"].(string)
				title, _ := rm["title"].(string)
				ch <- languagemodel.SourceURL{
					ID:    genID(),
					URL:   url,
					Title: strPtr(title),
					ProviderMetadata: shared.ProviderMetadata{
						"anthropic": jsonvalue.JSONObject{
							"pageAge": rm["page_age"],
						},
					},
				}
			}
		}
	} else if contentMap, ok := contentRaw.(map[string]any); ok {
		isErr := true
		errorCode, _ := contentMap["error_code"].(string)
		ch <- languagemodel.ToolResult{
			ToolCallID: toolUseID,
			ToolName:   toolNameMapping.ToCustomToolName("web_search"),
			IsError:    &isErr,
			Result: map[string]any{
				"type":      "web_search_tool_result_error",
				"errorCode": errorCode,
			},
		}
	}
}

// processCodeExecutionStreamResult processes code execution results in streaming.
func processCodeExecutionStreamResult(
	contentData map[string]any,
	toolUseID string,
	toolNameMapping providerutils.ToolNameMapping,
	ch chan<- languagemodel.StreamPart,
) {
	contentType, _ := contentData["type"].(string)
	switch contentType {
	case "code_execution_result":
		ch <- languagemodel.ToolResult{
			ToolCallID: toolUseID,
			ToolName:   toolNameMapping.ToCustomToolName("code_execution"),
			Result: map[string]any{
				"type":        contentType,
				"stdout":      contentData["stdout"],
				"stderr":      contentData["stderr"],
				"return_code": contentData["return_code"],
				"content":     defaultSlice(contentData["content"]),
			},
		}
	case "encrypted_code_execution_result":
		ch <- languagemodel.ToolResult{
			ToolCallID: toolUseID,
			ToolName:   toolNameMapping.ToCustomToolName("code_execution"),
			Result: map[string]any{
				"type":             contentType,
				"encrypted_stdout": contentData["encrypted_stdout"],
				"stderr":           contentData["stderr"],
				"return_code":      contentData["return_code"],
				"content":          defaultSlice(contentData["content"]),
			},
		}
	case "code_execution_tool_result_error":
		isErr := true
		ch <- languagemodel.ToolResult{
			ToolCallID: toolUseID,
			ToolName:   toolNameMapping.ToCustomToolName("code_execution"),
			IsError:    &isErr,
			Result: map[string]any{
				"type":      "code_execution_tool_result_error",
				"errorCode": contentData["error_code"],
			},
		}
	}
}

// processToolSearchStreamResult processes tool search results in streaming.
func processToolSearchStreamResult(
	contentData map[string]any,
	toolUseID string,
	serverToolCallsMap map[string]string,
	toolNameMapping providerutils.ToolNameMapping,
	ch chan<- languagemodel.StreamPart,
) {
	providerToolName := serverToolCallsMap[toolUseID]
	if providerToolName == "" {
		bm25Custom := toolNameMapping.ToCustomToolName("tool_search_tool_bm25")
		regexCustom := toolNameMapping.ToCustomToolName("tool_search_tool_regex")
		if bm25Custom != "tool_search_tool_bm25" {
			providerToolName = "tool_search_tool_bm25"
		} else if regexCustom != "tool_search_tool_regex" {
			providerToolName = "tool_search_tool_regex"
		} else {
			providerToolName = "tool_search_tool_regex"
		}
	}

	contentType, _ := contentData["type"].(string)
	if contentType == "tool_search_tool_search_result" {
		refs, _ := contentData["tool_references"].([]any)
		results := make([]map[string]any, 0, len(refs))
		for _, r := range refs {
			if rm, ok := r.(map[string]any); ok {
				results = append(results, map[string]any{
					"type":     rm["type"],
					"toolName": rm["tool_name"],
				})
			}
		}
		ch <- languagemodel.ToolResult{
			ToolCallID: toolUseID,
			ToolName:   toolNameMapping.ToCustomToolName(providerToolName),
			Result:     results,
		}
	} else {
		isErr := true
		ch <- languagemodel.ToolResult{
			ToolCallID: toolUseID,
			ToolName:   toolNameMapping.ToCustomToolName(providerToolName),
			IsError:    &isErr,
			Result: map[string]any{
				"type":      "tool_search_tool_result_error",
				"errorCode": contentData["error_code"],
			},
		}
	}
}

// --- Utility helpers ---

func strPtr(s string) *string { return &s }

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefStrPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func anyFromIntPtr(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func anyFromStrPtr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func isJsonFormatWithSchema(rf languagemodel.ResponseFormat) bool {
	if jsonFmt, ok := rf.(languagemodel.ResponseFormatJSON); ok {
		return jsonFmt.Schema != nil
	}
	return false
}

func defaultSlice(v any) any {
	if v == nil {
		return []any{}
	}
	return v
}

func usageToMap(usage AnthropicMessagesUsage) jsonvalue.JSONObject {
	m := jsonvalue.JSONObject{
		"input_tokens":  usage.InputTokens,
		"output_tokens": usage.OutputTokens,
	}
	if usage.CacheCreationInputTokens != nil {
		m["cache_creation_input_tokens"] = *usage.CacheCreationInputTokens
	}
	if usage.CacheReadInputTokens != nil {
		m["cache_read_input_tokens"] = *usage.CacheReadInputTokens
	}
	return m
}

func containerInfoToMap(c *AnthropicContainerInfo) map[string]any {
	if c == nil {
		return nil
	}
	m := map[string]any{
		"expiresAt": c.ExpiresAt,
		"id":        c.ID,
	}
	if c.Skills != nil {
		skills := make([]any, 0, len(c.Skills))
		for _, sk := range c.Skills {
			skills = append(skills, map[string]any{
				"type":    sk.Type,
				"skillId": sk.SkillID,
				"version": sk.Version,
			})
		}
		m["skills"] = skills
	} else {
		m["skills"] = nil
	}
	return m
}

func contextManagementRespToMap(cm *AnthropicContextManagementResponse) map[string]any {
	if cm == nil {
		return nil
	}
	return map[string]any{
		"appliedEdits": cm.AppliedEdits,
	}
}

// parseAnthropicOptions parses AnthropicLanguageModelOptions from raw provider options.
func parseAnthropicOptions(raw any) *AnthropicLanguageModelOptions {
	if raw == nil {
		return nil
	}

	// Try direct type assertion first
	if opts, ok := raw.(*AnthropicLanguageModelOptions); ok {
		return opts
	}
	if opts, ok := raw.(AnthropicLanguageModelOptions); ok {
		return &opts
	}

	// Try to parse from map
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	opts := &AnthropicLanguageModelOptions{}

	if v, ok := m["sendReasoning"].(bool); ok {
		opts.SendReasoning = &v
	}
	if v, ok := m["structuredOutputMode"].(string); ok {
		opts.StructuredOutputMode = &v
	}
	if v, ok := m["disableParallelToolUse"].(bool); ok {
		opts.DisableParallelToolUse = &v
	}
	if v, ok := m["toolStreaming"].(bool); ok {
		opts.ToolStreaming = &v
	}
	if v, ok := m["effort"].(string); ok {
		opts.Effort = &v
	}
	if v, ok := m["speed"].(string); ok {
		opts.Speed = &v
	}
	if v, ok := m["anthropicBeta"].([]any); ok {
		for _, b := range v {
			if s, ok := b.(string); ok {
				opts.AnthropicBeta = append(opts.AnthropicBeta, s)
			}
		}
	}

	// Thinking
	if thinkingRaw, ok := m["thinking"].(map[string]any); ok {
		thinking := &AnthropicThinkingConfig{}
		if t, ok := thinkingRaw["type"].(string); ok {
			thinking.Type = t
		}
		if bt, ok := thinkingRaw["budgetTokens"].(float64); ok {
			btInt := int(bt)
			thinking.BudgetTokens = &btInt
		}
		opts.Thinking = thinking
	}

	// Cache control
	if ccRaw, ok := m["cacheControl"].(map[string]any); ok {
		cc := &AnthropicCacheControl{}
		if t, ok := ccRaw["type"].(string); ok {
			cc.Type = t
		}
		if ttl, ok := ccRaw["ttl"].(string); ok {
			cc.TTL = &ttl
		}
		opts.CacheControl = cc
	}

	// Container
	if containerRaw, ok := m["container"].(map[string]any); ok {
		container := &AnthropicContainerConfig{}
		if id, ok := containerRaw["id"].(string); ok {
			container.ID = &id
		}
		if skillsRaw, ok := containerRaw["skills"].([]any); ok {
			for _, sr := range skillsRaw {
				if sm, ok := sr.(map[string]any); ok {
					skill := AnthropicContainerSkillConfig{}
					if t, ok := sm["type"].(string); ok {
						skill.Type = t
					}
					if sid, ok := sm["skillId"].(string); ok {
						skill.SkillID = sid
					}
					if v, ok := sm["version"].(string); ok {
						skill.Version = &v
					}
					container.Skills = append(container.Skills, skill)
				}
			}
		}
		opts.Container = container
	}

	// MCP servers
	if mcpRaw, ok := m["mcpServers"].([]any); ok {
		for _, sr := range mcpRaw {
			if sm, ok := sr.(map[string]any); ok {
				server := AnthropicMCPServerConfig{}
				if t, ok := sm["type"].(string); ok {
					server.Type = t
				}
				if n, ok := sm["name"].(string); ok {
					server.Name = n
				}
				if u, ok := sm["url"].(string); ok {
					server.URL = u
				}
				if at, ok := sm["authorizationToken"].(string); ok {
					server.AuthorizationToken = &at
				}
				if tcRaw, ok := sm["toolConfiguration"].(map[string]any); ok {
					tc := &AnthropicMCPToolConfiguration{}
					if enabled, ok := tcRaw["enabled"].(bool); ok {
						tc.Enabled = &enabled
					}
					if at, ok := tcRaw["allowedTools"].([]any); ok {
						for _, t := range at {
							if s, ok := t.(string); ok {
								tc.AllowedTools = append(tc.AllowedTools, s)
							}
						}
					}
					server.ToolConfiguration = tc
				}
				opts.MCPServers = append(opts.MCPServers, server)
			}
		}
	}

	// Context management
	if cmRaw, ok := m["contextManagement"].(map[string]any); ok {
		cm := &AnthropicContextManagementConfig{}
		if editsRaw, ok := cmRaw["edits"].([]any); ok {
			for _, er := range editsRaw {
				if em, ok := er.(map[string]any); ok {
					edit := AnthropicContextManagementEdit{}
					if t, ok := em["type"].(string); ok {
						edit.Type = t
					}
					if triggerRaw, ok := em["trigger"].(map[string]any); ok {
						trigger := &AnthropicContextManagementTrigger{}
						if t, ok := triggerRaw["type"].(string); ok {
							trigger.Type = t
						}
						if v, ok := triggerRaw["value"].(float64); ok {
							trigger.Value = int(v)
						}
						edit.Trigger = trigger
					}
					if keepRaw, ok := em["keep"].(map[string]any); ok {
						keep := &AnthropicContextManagementKeep{}
						if t, ok := keepRaw["type"].(string); ok {
							keep.Type = t
						}
						if v, ok := keepRaw["value"].(float64); ok {
							keep.Value = int(v)
						}
						edit.Keep = keep
					}
					if clearAtLeastRaw, ok := em["clearAtLeast"].(map[string]any); ok {
						cal := &AnthropicContextManagementClearAtLeast{}
						if t, ok := clearAtLeastRaw["type"].(string); ok {
							cal.Type = t
						}
						if v, ok := clearAtLeastRaw["value"].(float64); ok {
							cal.Value = int(v)
						}
						edit.ClearAtLeast = cal
					}
					if v, ok := em["clearToolInputs"].(bool); ok {
						edit.ClearToolInputs = &v
					}
					if et, ok := em["excludeTools"].([]any); ok {
						for _, t := range et {
							if s, ok := t.(string); ok {
								edit.ExcludeTools = append(edit.ExcludeTools, s)
							}
						}
					}
					if v, ok := em["pauseAfterCompaction"].(bool); ok {
						edit.PauseAfterCompaction = &v
					}
					if v, ok := em["instructions"].(string); ok {
						edit.Instructions = &v
					}
					cm.Edits = append(cm.Edits, edit)
				}
			}
		}
		opts.ContextManagement = cm
	}

	return opts
}

// mergeAnthropicOptions merges custom provider options into base options.
func mergeAnthropicOptions(base, custom *AnthropicLanguageModelOptions) {
	if custom.SendReasoning != nil {
		base.SendReasoning = custom.SendReasoning
	}
	if custom.StructuredOutputMode != nil {
		base.StructuredOutputMode = custom.StructuredOutputMode
	}
	if custom.Thinking != nil {
		base.Thinking = custom.Thinking
	}
	if custom.DisableParallelToolUse != nil {
		base.DisableParallelToolUse = custom.DisableParallelToolUse
	}
	if custom.CacheControl != nil {
		base.CacheControl = custom.CacheControl
	}
	if custom.MCPServers != nil {
		base.MCPServers = custom.MCPServers
	}
	if custom.Container != nil {
		base.Container = custom.Container
	}
	if custom.ToolStreaming != nil {
		base.ToolStreaming = custom.ToolStreaming
	}
	if custom.Effort != nil {
		base.Effort = custom.Effort
	}
	if custom.Speed != nil {
		base.Speed = custom.Speed
	}
	if custom.AnthropicBeta != nil {
		base.AnthropicBeta = custom.AnthropicBeta
	}
	if custom.ContextManagement != nil {
		base.ContextManagement = custom.ContextManagement
	}
}

// wrapErrorResponseHandler wraps a ResponseHandler[*APICallError] into a ResponseHandler[error].
func wrapErrorResponseHandler(h providerutils.ResponseHandler[*providerutils.APICallError]) providerutils.ResponseHandler[error] {
	return func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
		result, err := h(opts)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return &providerutils.ResponseHandlerResult[error]{
			Value:           result.Value,
			RawValue:        result.RawValue,
			ResponseHeaders: result.ResponseHeaders,
		}, nil
	}
}
