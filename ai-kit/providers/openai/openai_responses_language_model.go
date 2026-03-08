// Ported from: packages/openai/src/responses/openai-responses-language-model.ts
package openai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAIResponsesModelID represents known Responses API model identifiers.
// Any string is accepted to allow new models.
type OpenAIResponsesModelID = string

// extractApprovalRequestIDToToolCallIDMapping extracts a mapping from MCP approval
// request IDs to their corresponding tool call IDs from the prompt. When an MCP tool
// requires approval, we generate a tool call ID to track the pending approval in our
// system. When the user responds to the approval (and we continue the conversation),
// we need to map the approval request ID back to our tool call ID so that tool results
// reference the correct tool call.
func extractApprovalRequestIDToToolCallIDMapping(prompt languagemodel.Prompt) map[string]string {
	mapping := make(map[string]string)
	for _, message := range prompt {
		assistantMsg, ok := message.(languagemodel.AssistantMessage)
		if !ok {
			continue
		}
		for _, part := range assistantMsg.Content {
			toolCallPart, ok := part.(languagemodel.ToolCallPart)
			if !ok {
				continue
			}
			if toolCallPart.ProviderOptions == nil {
				continue
			}
			openaiOpts, ok := toolCallPart.ProviderOptions["openai"]
			if !ok {
				continue
			}
			approvalRequestID, ok := openaiOpts["approvalRequestId"].(string)
			if !ok || approvalRequestID == "" {
				continue
			}
			mapping[approvalRequestID] = toolCallPart.ToolCallID
		}
	}
	return mapping
}

// OpenAIResponsesLanguageModel implements languagemodel.LanguageModel for the
// OpenAI Responses API endpoint.
type OpenAIResponsesLanguageModel struct {
	modelID string
	config  OpenAIConfig
}

// NewOpenAIResponsesLanguageModel creates a new OpenAIResponsesLanguageModel.
func NewOpenAIResponsesLanguageModel(modelID OpenAIResponsesModelID, config OpenAIConfig) *OpenAIResponsesLanguageModel {
	return &OpenAIResponsesLanguageModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *OpenAIResponsesLanguageModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *OpenAIResponsesLanguageModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *OpenAIResponsesLanguageModel) ModelID() string { return m.modelID }

// SupportedUrls returns supported URL patterns.
func (m *OpenAIResponsesLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return map[string][]*regexp.Regexp{
		"image/*":          {regexp.MustCompile(`^https?://.*$`)},
		"application/pdf":  {regexp.MustCompile(`^https?://.*$`)},
	}, nil
}

// getArgsResult holds the result of getArgs.
type getArgsResult struct {
	webSearchToolName      string
	args                   map[string]any
	warnings               []shared.Warning
	store                  *bool
	toolNameMapping        providerutils.ToolNameMapping
	providerOptionsName    string
	isShellProviderExecuted bool
}

// getArgs builds the request arguments for the Responses API.
func (m *OpenAIResponsesLanguageModel) getArgs(opts languagemodel.CallOptions) (*getArgsResult, error) {
	var warnings []shared.Warning
	modelCapabilities := GetOpenAILanguageModelCapabilities(m.modelID)

	if opts.TopK != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "topK"})
	}
	if opts.Seed != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "seed"})
	}
	if opts.PresencePenalty != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "presencePenalty"})
	}
	if opts.FrequencyPenalty != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "frequencyPenalty"})
	}
	if len(opts.StopSequences) > 0 {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "stopSequences"})
	}

	providerOptionsName := "openai"
	if strings.Contains(m.config.Provider, "azure") {
		providerOptionsName = "azure"
	}

	openaiOptions := parseResponsesProviderOptions(providerOptionsName, opts.ProviderOptions)
	if openaiOptions == nil && providerOptionsName != "openai" {
		openaiOptions = parseResponsesProviderOptions("openai", opts.ProviderOptions)
	}

	isReasoningModel := modelCapabilities.IsReasoningModel
	if openaiOptions != nil && openaiOptions.ForceReasoning != nil {
		isReasoningModel = *openaiOptions.ForceReasoning
	}

	if openaiOptions != nil && openaiOptions.Conversation != nil && openaiOptions.PreviousResponseID != nil {
		details := "conversation and previousResponseId cannot be used together"
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "conversation", Details: &details})
	}

	// Build tool definitions for CreateToolNameMapping
	var toolDefs []providerutils.ProviderToolDefinition
	if opts.Tools != nil {
		for _, tool := range opts.Tools {
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
	}

	toolNameMapping := providerutils.CreateToolNameMapping(providerutils.CreateToolNameMappingOptions{
		Tools: toolDefs,
		ProviderToolNames: map[string]string{
			"openai.code_interpreter":  "code_interpreter",
			"openai.file_search":       "file_search",
			"openai.image_generation":  "image_generation",
			"openai.local_shell":       "local_shell",
			"openai.shell":             "shell",
			"openai.web_search":        "web_search",
			"openai.web_search_preview": "web_search_preview",
			"openai.mcp":               "mcp",
			"openai.apply_patch":        "apply_patch",
		},
		ResolveProviderToolName: func(tool providerutils.ProviderToolDefinition) *string {
			if tool.ID == "openai.custom" {
				// Find the actual provider tool to get the name from its args
				for _, t := range opts.Tools {
					if pt, ok := t.(languagemodel.ProviderTool); ok && pt.ID == "openai.custom" && pt.Name == tool.Name {
						if name, ok := pt.Args["name"].(string); ok {
							return &name
						}
					}
				}
			}
			return nil
		},
	})

	customProviderToolNames := make(map[string]struct{})
	preparedTools, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
		Tools:                   opts.Tools,
		ToolChoice:              opts.ToolChoice,
		ToolNameMapping:         &toolNameMapping,
		CustomProviderToolNames: customProviderToolNames,
	})
	if err != nil {
		return nil, err
	}

	hasOpenAITool := func(id string) bool {
		if opts.Tools == nil {
			return false
		}
		for _, tool := range opts.Tools {
			if pt, ok := tool.(languagemodel.ProviderTool); ok && pt.ID == id {
				return true
			}
		}
		return false
	}

	systemMessageMode := modelCapabilities.SystemMessageMode
	if openaiOptions != nil && openaiOptions.SystemMessageMode != nil {
		systemMessageMode = *openaiOptions.SystemMessageMode
	} else if isReasoningModel {
		systemMessageMode = "developer"
	}

	storeVal := true
	if openaiOptions != nil && openaiOptions.Store != nil {
		storeVal = *openaiOptions.Store
	}
	hasConversation := openaiOptions != nil && openaiOptions.Conversation != nil

	var customToolNamesPtr map[string]struct{}
	if len(customProviderToolNames) > 0 {
		customToolNamesPtr = customProviderToolNames
	}

	inputResult, err := ConvertToOpenAIResponsesInput(ConvertToOpenAIResponsesInputOptions{
		Prompt:                 opts.Prompt,
		ToolNameMapping:        toolNameMapping,
		SystemMessageMode:      systemMessageMode,
		ProviderOptionsName:    providerOptionsName,
		FileIDPrefixes:         m.config.FileIDPrefixes,
		Store:                  storeVal,
		HasConversation:        hasConversation,
		HasLocalShellTool:      hasOpenAITool("openai.local_shell"),
		HasShellTool:           hasOpenAITool("openai.shell"),
		HasApplyPatchTool:      hasOpenAITool("openai.apply_patch"),
		CustomProviderToolNames: customToolNamesPtr,
	})
	if err != nil {
		return nil, err
	}

	warnings = append(warnings, inputResult.Warnings...)

	strictJsonSchema := true
	if openaiOptions != nil && openaiOptions.StrictJSONSchema != nil {
		strictJsonSchema = *openaiOptions.StrictJSONSchema
	}

	var include []string
	if openaiOptions != nil {
		include = openaiOptions.Include
	}

	addInclude := func(key string) {
		if include == nil {
			include = []string{key}
		} else {
			for _, v := range include {
				if v == key {
					return
				}
			}
			include = append(include, key)
		}
	}

	// when logprobs are requested, automatically include them
	var topLogprobs *int
	if openaiOptions != nil && openaiOptions.Logprobs != nil {
		topLogprobs = openaiOptions.Logprobs.TopLogprobs()
	}
	if topLogprobs != nil {
		addInclude("message.output_text.logprobs")
	}

	// when a web search tool is present, automatically include the sources
	var webSearchToolName string
	if opts.Tools != nil {
		for _, tool := range opts.Tools {
			if pt, ok := tool.(languagemodel.ProviderTool); ok {
				if pt.ID == "openai.web_search" || pt.ID == "openai.web_search_preview" {
					webSearchToolName = pt.Name
					break
				}
			}
		}
	}
	if webSearchToolName != "" {
		addInclude("web_search_call.action.sources")
	}

	// when a code interpreter tool is present, automatically include the outputs
	if hasOpenAITool("openai.code_interpreter") {
		addInclude("code_interpreter_call.outputs")
	}

	var store *bool
	if openaiOptions != nil && openaiOptions.Store != nil {
		store = openaiOptions.Store
	}

	// store defaults to true in the OpenAI responses API, so check for false exactly
	if store != nil && !*store && isReasoningModel {
		addInclude("reasoning.encrypted_content")
	}

	// Build base args
	baseArgs := map[string]any{
		"model": m.modelID,
		"input": inputResult.Input,
	}

	setIfNotNil(baseArgs, "temperature", opts.Temperature)
	setIfNotNil(baseArgs, "top_p", opts.TopP)
	setIfNotNil(baseArgs, "max_output_tokens", opts.MaxOutputTokens)

	// text format settings
	hasTextSettings := false
	textSettings := map[string]any{}

	if opts.ResponseFormat != nil {
		if jsonFmt, ok := opts.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
			hasTextSettings = true
			if jsonFmt.Schema != nil {
				name := "response"
				if jsonFmt.Name != nil {
					name = *jsonFmt.Name
				}
				formatObj := map[string]any{
					"type":   "json_schema",
					"strict": strictJsonSchema,
					"name":   name,
					"schema": jsonFmt.Schema,
				}
				if jsonFmt.Description != nil {
					formatObj["description"] = *jsonFmt.Description
				}
				textSettings["format"] = formatObj
			} else {
				textSettings["format"] = map[string]any{"type": "json_object"}
			}
		}
	}

	if openaiOptions != nil && openaiOptions.TextVerbosity != nil {
		hasTextSettings = true
		textSettings["verbosity"] = *openaiOptions.TextVerbosity
	}

	if hasTextSettings {
		baseArgs["text"] = textSettings
	}

	// provider options
	if openaiOptions != nil {
		setIfNotNil(baseArgs, "conversation", openaiOptions.Conversation)
		setIfNotNil(baseArgs, "max_tool_calls", openaiOptions.MaxToolCalls)
		if openaiOptions.Metadata != nil {
			baseArgs["metadata"] = openaiOptions.Metadata
		}
		setIfNotNil(baseArgs, "parallel_tool_calls", openaiOptions.ParallelToolCalls)
		setIfNotNil(baseArgs, "previous_response_id", openaiOptions.PreviousResponseID)
		if openaiOptions.Store != nil {
			baseArgs["store"] = *openaiOptions.Store
		}
		setIfNotNil(baseArgs, "user", openaiOptions.User)
		setIfNotNil(baseArgs, "instructions", openaiOptions.Instructions)
		setIfNotNil(baseArgs, "service_tier", openaiOptions.ServiceTier)
		setIfNotNil(baseArgs, "prompt_cache_key", openaiOptions.PromptCacheKey)
		setIfNotNil(baseArgs, "prompt_cache_retention", openaiOptions.PromptCacheRetention)
		setIfNotNil(baseArgs, "safety_identifier", openaiOptions.SafetyIdentifier)
		setIfNotNil(baseArgs, "truncation", openaiOptions.Truncation)
	}

	if include != nil {
		baseArgs["include"] = include
	}
	if topLogprobs != nil {
		baseArgs["top_logprobs"] = *topLogprobs
	}

	// model-specific reasoning settings
	if isReasoningModel && openaiOptions != nil {
		if openaiOptions.ReasoningEffort != nil || openaiOptions.ReasoningSummary != nil {
			reasoning := map[string]any{}
			if openaiOptions.ReasoningEffort != nil {
				reasoning["effort"] = *openaiOptions.ReasoningEffort
			}
			if openaiOptions.ReasoningSummary != nil {
				reasoning["summary"] = *openaiOptions.ReasoningSummary
			}
			baseArgs["reasoning"] = reasoning
		}
	}

	// remove unsupported settings for reasoning models
	if isReasoningModel {
		allowNonReasoning := openaiOptions != nil &&
			openaiOptions.ReasoningEffort != nil &&
			*openaiOptions.ReasoningEffort == "none" &&
			modelCapabilities.SupportsNonReasoningParameters

		if !allowNonReasoning {
			if _, ok := baseArgs["temperature"]; ok {
				delete(baseArgs, "temperature")
				details := "temperature is not supported for reasoning models"
				warnings = append(warnings, shared.UnsupportedWarning{Feature: "temperature", Details: &details})
			}
			if _, ok := baseArgs["top_p"]; ok {
				delete(baseArgs, "top_p")
				details := "topP is not supported for reasoning models"
				warnings = append(warnings, shared.UnsupportedWarning{Feature: "topP", Details: &details})
			}
		}
	} else {
		if openaiOptions != nil && openaiOptions.ReasoningEffort != nil {
			details := "reasoningEffort is not supported for non-reasoning models"
			warnings = append(warnings, shared.UnsupportedWarning{Feature: "reasoningEffort", Details: &details})
		}
		if openaiOptions != nil && openaiOptions.ReasoningSummary != nil {
			details := "reasoningSummary is not supported for non-reasoning models"
			warnings = append(warnings, shared.UnsupportedWarning{Feature: "reasoningSummary", Details: &details})
		}
	}

	// Validate flex processing support
	if openaiOptions != nil && openaiOptions.ServiceTier != nil && *openaiOptions.ServiceTier == "flex" && !modelCapabilities.SupportsFlexProcessing {
		details := "flex processing is only available for o3, o4-mini, and gpt-5 models"
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "serviceTier", Details: &details})
		delete(baseArgs, "service_tier")
	}

	// Validate priority processing support
	if openaiOptions != nil && openaiOptions.ServiceTier != nil && *openaiOptions.ServiceTier == "priority" && !modelCapabilities.SupportsPriorityProcessing {
		details := "priority processing is only available for supported models (gpt-4, gpt-5, gpt-5-mini, o3, o4-mini) and requires Enterprise access. gpt-5-nano is not supported"
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "serviceTier", Details: &details})
		delete(baseArgs, "service_tier")
	}

	// Determine if shell tool uses provider-executed environment
	isShellProviderExecuted := false
	if opts.Tools != nil {
		for _, tool := range opts.Tools {
			if pt, ok := tool.(languagemodel.ProviderTool); ok && pt.ID == "openai.shell" {
				if env, ok := pt.Args["environment"].(map[string]any); ok {
					envType, _ := env["type"].(string)
					if envType == "containerAuto" || envType == "containerReference" {
						isShellProviderExecuted = true
					}
				}
				break
			}
		}
	}

	// Combine base args with tool settings
	baseArgs["tools"] = preparedTools.Tools
	baseArgs["tool_choice"] = preparedTools.ToolChoice

	warnings = append(warnings, preparedTools.ToolWarnings...)

	return &getArgsResult{
		webSearchToolName:       webSearchToolName,
		args:                    baseArgs,
		warnings:                warnings,
		store:                   store,
		toolNameMapping:         toolNameMapping,
		providerOptionsName:     providerOptionsName,
		isShellProviderExecuted: isShellProviderExecuted,
	}, nil
}

// DoGenerate implements the non-streaming generation for the Responses API.
func (m *OpenAIResponsesLanguageModel) DoGenerate(opts languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	argsResult, err := m.getArgs(opts)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}
	body := argsResult.args
	url := m.config.URL(struct {
		ModelID string
		Path    string
	}{ModelID: m.modelID, Path: "/responses"})

	approvalRequestIDToDummyToolCallIDFromPrompt := extractApprovalRequestIDToToolCallIDMapping(opts.Prompt)

	apiResult, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[OpenAIResponsesResponse]{
		URL:                   url,
		Headers:               providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(opts.Headers)),
		Body:                  body,
		FailedResponseHandler: openaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(
			openaiResponsesResponseSchema,
		),
		Ctx:   opts.Ctx,
		Fetch: m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	response := apiResult.Value
	responseHeaders := apiResult.ResponseHeaders

	if response.Error != nil {
		return languagemodel.GenerateResult{}, providerutils.NewAPICallError(providerutils.APICallErrorOptions{
			Message:           response.Error.Message,
			URL:               url,
			RequestBodyValues: body,
			StatusCode:        400,
			ResponseHeaders:   responseHeaders,
		})
	}

	var content []languagemodel.Content
	var logprobs [][]OpenAIResponsesLogprob
	hasFunctionCall := false

	genID := m.generateID()

	for _, part := range response.Output {
		partType, _ := part["type"].(string)
		switch partType {
		case "reasoning":
			summaries := getMapSlice(part, "summary")
			if len(summaries) == 0 {
				summaries = []map[string]any{{"type": "summary_text", "text": ""}}
			}
			partID, _ := part["id"].(string)
			encryptedContent, _ := part["encrypted_content"].(string)
			for _, summary := range summaries {
				summaryText, _ := summary["text"].(string)
				content = append(content, languagemodel.Reasoning{
					Text: summaryText,
					ProviderMetadata: shared.ProviderMetadata{
						argsResult.providerOptionsName: map[string]any{
							"itemId":                      partID,
							"reasoningEncryptedContent":   nilIfEmpty(encryptedContent),
						},
					},
				})
			}

		case "image_generation_call":
			partID, _ := part["id"].(string)
			result, _ := part["result"]
			content = append(content, languagemodel.ToolCall{
				ToolCallID:       partID,
				ToolName:         argsResult.toolNameMapping.ToCustomToolName("image_generation"),
				Input:            "{}",
				ProviderExecuted: boolPtr(true),
			})
			content = append(content, languagemodel.ToolResult{
				ToolCallID: partID,
				ToolName:   argsResult.toolNameMapping.ToCustomToolName("image_generation"),
				Result:     map[string]any{"result": result},
			})

		case "local_shell_call":
			callID, _ := part["call_id"].(string)
			partID, _ := part["id"].(string)
			action := part["action"]
			content = append(content, languagemodel.ToolCall{
				ToolCallID: callID,
				ToolName:   argsResult.toolNameMapping.ToCustomToolName("local_shell"),
				Input:      jsonString(map[string]any{"action": action}),
				ProviderMetadata: shared.ProviderMetadata{
					argsResult.providerOptionsName: map[string]any{"itemId": partID},
				},
			})

		case "shell_call":
			callID, _ := part["call_id"].(string)
			partID, _ := part["id"].(string)
			action, _ := part["action"].(map[string]any)
			commands, _ := action["commands"]
			inputData := map[string]any{"action": map[string]any{"commands": commands}}
			tc := languagemodel.ToolCall{
				ToolCallID: callID,
				ToolName:   argsResult.toolNameMapping.ToCustomToolName("shell"),
				Input:      jsonString(inputData),
				ProviderMetadata: shared.ProviderMetadata{
					argsResult.providerOptionsName: map[string]any{"itemId": partID},
				},
			}
			if argsResult.isShellProviderExecuted {
				tc.ProviderExecuted = boolPtr(true)
			}
			content = append(content, tc)

		case "shell_call_output":
			callID, _ := part["call_id"].(string)
			outputItems := getAnySlice(part, "output")
			content = append(content, languagemodel.ToolResult{
				ToolCallID: callID,
				ToolName:   argsResult.toolNameMapping.ToCustomToolName("shell"),
				Result:     map[string]any{"output": convertShellOutputItems(outputItems)},
			})

		case "message":
			partID, _ := part["id"].(string)
			phase, _ := part["phase"].(string)
			contentParts := getAnySlice(part, "content")
			for _, cp := range contentParts {
				cpMap, ok := cp.(map[string]any)
				if !ok {
					continue
				}
				// Logprobs handling
				if opts.ProviderOptions != nil {
					if provOpts, ok := opts.ProviderOptions[argsResult.providerOptionsName]; ok {
						if provOpts["logprobs"] != nil {
							if lp, ok := cpMap["logprobs"].([]any); ok && len(lp) > 0 {
								logprobs = append(logprobs, convertLogprobs(lp))
							}
						}
					}
				}

				textVal, _ := cpMap["text"].(string)
				annotations := getAnySlice(cpMap, "annotations")

				provMeta := map[string]any{"itemId": partID}
				if phase != "" {
					provMeta["phase"] = phase
				}
				if len(annotations) > 0 {
					provMeta["annotations"] = annotations
				}

				content = append(content, languagemodel.Text{
					Text: textVal,
					ProviderMetadata: shared.ProviderMetadata{
						argsResult.providerOptionsName: provMeta,
					},
				})

				for _, ann := range annotations {
					annMap, ok := ann.(map[string]any)
					if !ok {
						continue
					}
					content = append(content, m.createSourceFromAnnotation(annMap, argsResult.providerOptionsName, genID)...)
				}
			}

		case "function_call":
			hasFunctionCall = true
			callID, _ := part["call_id"].(string)
			name, _ := part["name"].(string)
			arguments, _ := part["arguments"].(string)
			partID, _ := part["id"].(string)
			content = append(content, languagemodel.ToolCall{
				ToolCallID: callID,
				ToolName:   name,
				Input:      arguments,
				ProviderMetadata: shared.ProviderMetadata{
					argsResult.providerOptionsName: map[string]any{"itemId": partID},
				},
			})

		case "custom_tool_call":
			hasFunctionCall = true
			callID, _ := part["call_id"].(string)
			name, _ := part["name"].(string)
			input := part["input"]
			partID, _ := part["id"].(string)
			toolName := argsResult.toolNameMapping.ToCustomToolName(name)
			content = append(content, languagemodel.ToolCall{
				ToolCallID: callID,
				ToolName:   toolName,
				Input:      jsonString(input),
				ProviderMetadata: shared.ProviderMetadata{
					argsResult.providerOptionsName: map[string]any{"itemId": partID},
				},
			})

		case "web_search_call":
			partID, _ := part["id"].(string)
			action, _ := part["action"].(map[string]any)
			wsName := argsResult.webSearchToolName
			if wsName == "" {
				wsName = "web_search"
			}
			content = append(content, languagemodel.ToolCall{
				ToolCallID:       partID,
				ToolName:         argsResult.toolNameMapping.ToCustomToolName(wsName),
				Input:            "{}",
				ProviderExecuted: boolPtr(true),
			})
			content = append(content, languagemodel.ToolResult{
				ToolCallID: partID,
				ToolName:   argsResult.toolNameMapping.ToCustomToolName(wsName),
				Result:     mapWebSearchOutput(action),
			})

		case "mcp_call":
			partID, _ := part["id"].(string)
			approvalRequestID, _ := part["approval_request_id"].(string)
			toolCallID := partID
			if approvalRequestID != "" {
				if mapped, ok := approvalRequestIDToDummyToolCallIDFromPrompt[approvalRequestID]; ok {
					toolCallID = mapped
				}
			}
			name, _ := part["name"].(string)
			arguments, _ := part["arguments"].(string)
			serverLabel, _ := part["server_label"].(string)
			toolName := fmt.Sprintf("mcp.%s", name)

			content = append(content, languagemodel.ToolCall{
				ToolCallID:       toolCallID,
				ToolName:         toolName,
				Input:            arguments,
				ProviderExecuted: boolPtr(true),
				Dynamic:          boolPtr(true),
			})

			mcpResult := map[string]any{
				"type":        "call",
				"serverLabel": serverLabel,
				"name":        name,
				"arguments":   arguments,
			}
			if output, ok := part["output"]; ok && output != nil {
				mcpResult["output"] = output
			}
			if errVal, ok := part["error"]; ok && errVal != nil {
				mcpResult["error"] = errVal
			}
			content = append(content, languagemodel.ToolResult{
				ToolCallID: toolCallID,
				ToolName:   toolName,
				Result:     mcpResult,
				ProviderMetadata: shared.ProviderMetadata{
					argsResult.providerOptionsName: map[string]any{"itemId": partID},
				},
			})

		case "mcp_list_tools":
			// skip

		case "mcp_approval_request":
			approvalRequestID, _ := part["approval_request_id"].(string)
			partID, _ := part["id"].(string)
			if approvalRequestID == "" {
				approvalRequestID = partID
			}
			dummyToolCallID := genID()
			name, _ := part["name"].(string)
			arguments, _ := part["arguments"].(string)
			toolName := fmt.Sprintf("mcp.%s", name)

			content = append(content, languagemodel.ToolCall{
				ToolCallID:       dummyToolCallID,
				ToolName:         toolName,
				Input:            arguments,
				ProviderExecuted: boolPtr(true),
				Dynamic:          boolPtr(true),
			})
			content = append(content, languagemodel.ToolApprovalRequest{
				ApprovalID: approvalRequestID,
				ToolCallID: dummyToolCallID,
			})

		case "computer_call":
			partID, _ := part["id"].(string)
			status, _ := part["status"].(string)
			if status == "" {
				status = "completed"
			}
			content = append(content, languagemodel.ToolCall{
				ToolCallID:       partID,
				ToolName:         argsResult.toolNameMapping.ToCustomToolName("computer_use"),
				Input:            "",
				ProviderExecuted: boolPtr(true),
			})
			content = append(content, languagemodel.ToolResult{
				ToolCallID: partID,
				ToolName:   argsResult.toolNameMapping.ToCustomToolName("computer_use"),
				Result:     map[string]any{"type": "computer_use_tool_result", "status": status},
			})

		case "file_search_call":
			partID, _ := part["id"].(string)
			queries, _ := part["queries"]
			content = append(content, languagemodel.ToolCall{
				ToolCallID:       partID,
				ToolName:         argsResult.toolNameMapping.ToCustomToolName("file_search"),
				Input:            "{}",
				ProviderExecuted: boolPtr(true),
			})
			content = append(content, languagemodel.ToolResult{
				ToolCallID: partID,
				ToolName:   argsResult.toolNameMapping.ToCustomToolName("file_search"),
				Result: map[string]any{
					"queries": queries,
					"results": convertFileSearchResults(part),
				},
			})

		case "code_interpreter_call":
			partID, _ := part["id"].(string)
			code, _ := part["code"].(string)
			containerID, _ := part["container_id"].(string)
			outputs := part["outputs"]
			content = append(content, languagemodel.ToolCall{
				ToolCallID:       partID,
				ToolName:         argsResult.toolNameMapping.ToCustomToolName("code_interpreter"),
				Input:            jsonString(map[string]any{"code": code, "containerId": containerID}),
				ProviderExecuted: boolPtr(true),
			})
			content = append(content, languagemodel.ToolResult{
				ToolCallID: partID,
				ToolName:   argsResult.toolNameMapping.ToCustomToolName("code_interpreter"),
				Result:     map[string]any{"outputs": outputs},
			})

		case "apply_patch_call":
			callID, _ := part["call_id"].(string)
			partID, _ := part["id"].(string)
			operation := part["operation"]
			content = append(content, languagemodel.ToolCall{
				ToolCallID: callID,
				ToolName:   argsResult.toolNameMapping.ToCustomToolName("apply_patch"),
				Input:      jsonString(map[string]any{"callId": callID, "operation": operation}),
				ProviderMetadata: shared.ProviderMetadata{
					argsResult.providerOptionsName: map[string]any{"itemId": partID},
				},
			})
		}
	}

	providerMetadata := shared.ProviderMetadata{
		argsResult.providerOptionsName: map[string]any{
			"responseId": response.ID,
		},
	}
	if len(logprobs) > 0 {
		providerMetadata[argsResult.providerOptionsName]["logprobs"] = logprobs
	}
	if response.ServiceTier != nil {
		providerMetadata[argsResult.providerOptionsName]["serviceTier"] = *response.ServiceTier
	}

	var finishReasonRaw *string
	if response.IncompleteDetails != nil {
		finishReasonRaw = &response.IncompleteDetails.Reason
	}

	var timestamp *time.Time
	if response.CreatedAt != nil {
		t := time.Unix(*response.CreatedAt, 0)
		timestamp = &t
	}

	return languagemodel.GenerateResult{
		Content: content,
		FinishReason: languagemodel.FinishReason{
			Unified: MapOpenAIResponseFinishReason(MapOpenAIResponseFinishReasonOptions{
				FinishReason:    finishReasonRaw,
				HasFunctionCall: hasFunctionCall,
			}),
			Raw: finishReasonRaw,
		},
		Usage:            ConvertOpenAIResponsesUsage(response.Usage),
		Request:          &languagemodel.GenerateResultRequest{Body: body},
		Response: &languagemodel.GenerateResultResponse{
			ResponseMetadata: languagemodel.ResponseMetadata{
				ID:        &response.ID,
				Timestamp: timestamp,
				ModelID:   &response.Model,
			},
			Headers: responseHeaders,
			Body:    apiResult.RawValue,
		},
		ProviderMetadata: providerMetadata,
		Warnings:         argsResult.warnings,
	}, nil
}

// DoStream implements the streaming generation for the Responses API.
func (m *OpenAIResponsesLanguageModel) DoStream(opts languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	argsResult, err := m.getArgs(opts)
	if err != nil {
		return languagemodel.StreamResult{}, err
	}
	body := argsResult.args
	body["stream"] = true

	url := m.config.URL(struct {
		ModelID string
		Path    string
	}{ModelID: m.modelID, Path: "/responses"})

	apiResult, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[<-chan providerutils.ParseResult[OpenAIResponsesChunk]]{
		URL:                   url,
		Headers:               providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(opts.Headers)),
		Body:                  body,
		FailedResponseHandler: openaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(
			openaiResponsesChunkSchema,
		),
		Ctx:   opts.Ctx,
		Fetch: m.config.Fetch,
	})
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	responseHeaders := apiResult.ResponseHeaders
	eventStream := apiResult.Value

	approvalRequestIDToDummyToolCallIDFromPrompt := extractApprovalRequestIDToToolCallIDMapping(opts.Prompt)
	approvalRequestIDToDummyToolCallIDFromStream := make(map[string]string)

	outCh := make(chan languagemodel.StreamPart, 64)

	go func() {
		defer close(outCh)

		genID := m.generateID()

		finishReason := languagemodel.FinishReason{
			Unified: languagemodel.FinishReasonOther,
		}
		var usage *OpenAIResponsesUsage
		var logprobs [][]OpenAIResponsesLogprob
		var responseID string
		var serviceTier string

		// ongoing tool calls indexed by output_index
		type ongoingToolCallInfo struct {
			toolName    string
			toolCallID  string
			codeInterpreter *struct {
				containerID string
			}
			applyPatch *struct {
				hasDiff     bool
				endEmitted  bool
			}
		}
		ongoingToolCalls := make(map[int]*ongoingToolCallInfo)

		var ongoingAnnotations []map[string]any
		var activeMessagePhase string
		hasFunctionCall := false

		type reasoningState struct {
			encryptedContent *string
			summaryParts     map[string]string // "active", "can-conclude", "concluded"
		}
		activeReasoning := make(map[string]*reasoningState)

		// Send stream-start
		outCh <- languagemodel.StreamPartStreamStart{
			Warnings: argsResult.warnings,
		}

		for chunk := range eventStream {
			if opts.IncludeRawChunks != nil && *opts.IncludeRawChunks {
				outCh <- languagemodel.StreamPartRaw{RawValue: chunk.RawValue}
			}

			if !chunk.Success {
				finishReason = languagemodel.FinishReason{
					Unified: languagemodel.FinishReasonError,
				}
				outCh <- languagemodel.StreamPartError{Error: chunk.Error}
				continue
			}

			value := chunk.Value

			switch value.Type {
			case "response.output_item.added":
				itemType, _ := value.Item["type"].(string)

				switch itemType {
				case "function_call":
					itemName, _ := value.Item["name"].(string)
					itemCallID, _ := value.Item["call_id"].(string)
					ongoingToolCalls[value.OutputIndex] = &ongoingToolCallInfo{
						toolName:   itemName,
						toolCallID: itemCallID,
					}
					outCh <- languagemodel.StreamPartToolInputStart{
						ID:       itemCallID,
						ToolName: itemName,
					}

				case "custom_tool_call":
					itemName, _ := value.Item["name"].(string)
					itemCallID, _ := value.Item["call_id"].(string)
					toolName := argsResult.toolNameMapping.ToCustomToolName(itemName)
					ongoingToolCalls[value.OutputIndex] = &ongoingToolCallInfo{
						toolName:   toolName,
						toolCallID: itemCallID,
					}
					outCh <- languagemodel.StreamPartToolInputStart{
						ID:       itemCallID,
						ToolName: toolName,
					}

				case "web_search_call":
					itemID, _ := value.Item["id"].(string)
					wsName := argsResult.webSearchToolName
					if wsName == "" {
						wsName = "web_search"
					}
					toolName := argsResult.toolNameMapping.ToCustomToolName(wsName)
					ongoingToolCalls[value.OutputIndex] = &ongoingToolCallInfo{
						toolName:   toolName,
						toolCallID: itemID,
					}
					outCh <- languagemodel.StreamPartToolInputStart{
						ID:               itemID,
						ToolName:         toolName,
						ProviderExecuted: boolPtr(true),
					}
					outCh <- languagemodel.StreamPartToolInputEnd{ID: itemID}
					outCh <- languagemodel.ToolCall{
						ToolCallID:       itemID,
						ToolName:         toolName,
						Input:            "{}",
						ProviderExecuted: boolPtr(true),
					}

				case "computer_call":
					itemID, _ := value.Item["id"].(string)
					toolName := argsResult.toolNameMapping.ToCustomToolName("computer_use")
					ongoingToolCalls[value.OutputIndex] = &ongoingToolCallInfo{
						toolName:   toolName,
						toolCallID: itemID,
					}
					outCh <- languagemodel.StreamPartToolInputStart{
						ID:               itemID,
						ToolName:         toolName,
						ProviderExecuted: boolPtr(true),
					}

				case "code_interpreter_call":
					itemID, _ := value.Item["id"].(string)
					containerID, _ := value.Item["container_id"].(string)
					toolName := argsResult.toolNameMapping.ToCustomToolName("code_interpreter")
					ongoingToolCalls[value.OutputIndex] = &ongoingToolCallInfo{
						toolName:   toolName,
						toolCallID: itemID,
						codeInterpreter: &struct{ containerID string }{containerID: containerID},
					}
					outCh <- languagemodel.StreamPartToolInputStart{
						ID:               itemID,
						ToolName:         toolName,
						ProviderExecuted: boolPtr(true),
					}
					outCh <- languagemodel.StreamPartToolInputDelta{
						ID:    itemID,
						Delta: fmt.Sprintf(`{"containerId":"%s","code":"`, escapeJSONDelta(containerID)),
					}

				case "file_search_call":
					itemID, _ := value.Item["id"].(string)
					outCh <- languagemodel.ToolCall{
						ToolCallID:       itemID,
						ToolName:         argsResult.toolNameMapping.ToCustomToolName("file_search"),
						Input:            "{}",
						ProviderExecuted: boolPtr(true),
					}

				case "image_generation_call":
					itemID, _ := value.Item["id"].(string)
					outCh <- languagemodel.ToolCall{
						ToolCallID:       itemID,
						ToolName:         argsResult.toolNameMapping.ToCustomToolName("image_generation"),
						Input:            "{}",
						ProviderExecuted: boolPtr(true),
					}

				case "mcp_call", "mcp_list_tools", "mcp_approval_request":
					// handled in output_item.done

				case "apply_patch_call":
					callID, _ := value.Item["call_id"].(string)
					operation, _ := value.Item["operation"].(map[string]any)
					opType, _ := operation["type"].(string)

					isDelete := opType == "delete_file"
					ongoingToolCalls[value.OutputIndex] = &ongoingToolCallInfo{
						toolName:   argsResult.toolNameMapping.ToCustomToolName("apply_patch"),
						toolCallID: callID,
						applyPatch: &struct {
							hasDiff    bool
							endEmitted bool
						}{hasDiff: isDelete, endEmitted: isDelete},
					}

					outCh <- languagemodel.StreamPartToolInputStart{
						ID:       callID,
						ToolName: argsResult.toolNameMapping.ToCustomToolName("apply_patch"),
					}

					if isDelete {
						inputString := jsonString(map[string]any{"callId": callID, "operation": operation})
						outCh <- languagemodel.StreamPartToolInputDelta{ID: callID, Delta: inputString}
						outCh <- languagemodel.StreamPartToolInputEnd{ID: callID}
					} else {
						opPath, _ := operation["path"].(string)
						outCh <- languagemodel.StreamPartToolInputDelta{
							ID:    callID,
							Delta: fmt.Sprintf(`{"callId":"%s","operation":{"type":"%s","path":"%s","diff":"`, escapeJSONDelta(callID), escapeJSONDelta(opType), escapeJSONDelta(opPath)),
						}
					}

				case "shell_call":
					callID, _ := value.Item["call_id"].(string)
					ongoingToolCalls[value.OutputIndex] = &ongoingToolCallInfo{
						toolName:   argsResult.toolNameMapping.ToCustomToolName("shell"),
						toolCallID: callID,
					}

				case "shell_call_output":
					// handled in output_item.done

				case "message":
					ongoingAnnotations = nil
					itemID, _ := value.Item["id"].(string)
					phase, _ := value.Item["phase"].(string)
					activeMessagePhase = phase

					provMeta := shared.ProviderMetadata{
						argsResult.providerOptionsName: map[string]any{
							"itemId": itemID,
						},
					}
					if phase != "" {
						provMeta[argsResult.providerOptionsName]["phase"] = phase
					}
					outCh <- languagemodel.StreamPartTextStart{
						ID:               itemID,
						ProviderMetadata: provMeta,
					}

				case "reasoning":
					itemID, _ := value.Item["id"].(string)
					encryptedContent, _ := value.Item["encrypted_content"].(string)
					var encPtr *string
					if encryptedContent != "" {
						encPtr = &encryptedContent
					}
					activeReasoning[itemID] = &reasoningState{
						encryptedContent: encPtr,
						summaryParts:     map[string]string{"0": "active"},
					}
					outCh <- languagemodel.StreamPartReasoningStart{
						ID: fmt.Sprintf("%s:0", itemID),
						ProviderMetadata: shared.ProviderMetadata{
							argsResult.providerOptionsName: map[string]any{
								"itemId":                    itemID,
								"reasoningEncryptedContent": nilIfEmptyPtr(encPtr),
							},
						},
					}
				}

			case "response.output_item.done":
				itemType, _ := value.Item["type"].(string)

				switch itemType {
				case "message":
					itemID, _ := value.Item["id"].(string)
					phase, _ := value.Item["phase"].(string)
					if phase == "" {
						phase = activeMessagePhase
					}
					activeMessagePhase = ""
					provMeta := map[string]any{"itemId": itemID}
					if phase != "" {
						provMeta["phase"] = phase
					}
					if len(ongoingAnnotations) > 0 {
						provMeta["annotations"] = ongoingAnnotations
					}
					outCh <- languagemodel.StreamPartTextEnd{
						ID: itemID,
						ProviderMetadata: shared.ProviderMetadata{
							argsResult.providerOptionsName: provMeta,
						},
					}

				case "function_call":
					delete(ongoingToolCalls, value.OutputIndex)
					hasFunctionCall = true
					callID, _ := value.Item["call_id"].(string)
					name, _ := value.Item["name"].(string)
					arguments, _ := value.Item["arguments"].(string)
					itemID, _ := value.Item["id"].(string)
					outCh <- languagemodel.StreamPartToolInputEnd{ID: callID}
					outCh <- languagemodel.ToolCall{
						ToolCallID: callID,
						ToolName:   name,
						Input:      arguments,
						ProviderMetadata: shared.ProviderMetadata{
							argsResult.providerOptionsName: map[string]any{"itemId": itemID},
						},
					}

				case "custom_tool_call":
					delete(ongoingToolCalls, value.OutputIndex)
					hasFunctionCall = true
					callID, _ := value.Item["call_id"].(string)
					name, _ := value.Item["name"].(string)
					input := value.Item["input"]
					itemID, _ := value.Item["id"].(string)
					toolName := argsResult.toolNameMapping.ToCustomToolName(name)
					outCh <- languagemodel.StreamPartToolInputEnd{ID: callID}
					outCh <- languagemodel.ToolCall{
						ToolCallID: callID,
						ToolName:   toolName,
						Input:      jsonString(input),
						ProviderMetadata: shared.ProviderMetadata{
							argsResult.providerOptionsName: map[string]any{"itemId": itemID},
						},
					}

				case "web_search_call":
					delete(ongoingToolCalls, value.OutputIndex)
					itemID, _ := value.Item["id"].(string)
					action, _ := value.Item["action"].(map[string]any)
					wsName := argsResult.webSearchToolName
					if wsName == "" {
						wsName = "web_search"
					}
					outCh <- languagemodel.ToolResult{
						ToolCallID: itemID,
						ToolName:   argsResult.toolNameMapping.ToCustomToolName(wsName),
						Result:     mapWebSearchOutput(action),
					}

				case "computer_call":
					delete(ongoingToolCalls, value.OutputIndex)
					itemID, _ := value.Item["id"].(string)
					status, _ := value.Item["status"].(string)
					if status == "" {
						status = "completed"
					}
					outCh <- languagemodel.StreamPartToolInputEnd{ID: itemID}
					outCh <- languagemodel.ToolCall{
						ToolCallID:       itemID,
						ToolName:         argsResult.toolNameMapping.ToCustomToolName("computer_use"),
						Input:            "",
						ProviderExecuted: boolPtr(true),
					}
					outCh <- languagemodel.ToolResult{
						ToolCallID: itemID,
						ToolName:   argsResult.toolNameMapping.ToCustomToolName("computer_use"),
						Result:     map[string]any{"type": "computer_use_tool_result", "status": status},
					}

				case "file_search_call":
					delete(ongoingToolCalls, value.OutputIndex)
					itemID, _ := value.Item["id"].(string)
					queries := value.Item["queries"]
					outCh <- languagemodel.ToolResult{
						ToolCallID: itemID,
						ToolName:   argsResult.toolNameMapping.ToCustomToolName("file_search"),
						Result: map[string]any{
							"queries": queries,
							"results": convertFileSearchResults(value.Item),
						},
					}

				case "code_interpreter_call":
					delete(ongoingToolCalls, value.OutputIndex)
					itemID, _ := value.Item["id"].(string)
					outputs := value.Item["outputs"]
					outCh <- languagemodel.ToolResult{
						ToolCallID: itemID,
						ToolName:   argsResult.toolNameMapping.ToCustomToolName("code_interpreter"),
						Result:     map[string]any{"outputs": outputs},
					}

				case "image_generation_call":
					itemID, _ := value.Item["id"].(string)
					result := value.Item["result"]
					outCh <- languagemodel.ToolResult{
						ToolCallID: itemID,
						ToolName:   argsResult.toolNameMapping.ToCustomToolName("image_generation"),
						Result:     map[string]any{"result": result},
					}

				case "mcp_call":
					delete(ongoingToolCalls, value.OutputIndex)
					itemID, _ := value.Item["id"].(string)
					approvalRequestID, _ := value.Item["approval_request_id"].(string)

					aliasedToolCallID := itemID
					if approvalRequestID != "" {
						if mapped, ok := approvalRequestIDToDummyToolCallIDFromStream[approvalRequestID]; ok {
							aliasedToolCallID = mapped
						} else if mapped, ok := approvalRequestIDToDummyToolCallIDFromPrompt[approvalRequestID]; ok {
							aliasedToolCallID = mapped
						}
					}

					name, _ := value.Item["name"].(string)
					arguments, _ := value.Item["arguments"].(string)
					serverLabel, _ := value.Item["server_label"].(string)
					toolName := fmt.Sprintf("mcp.%s", name)

					outCh <- languagemodel.ToolCall{
						ToolCallID:       aliasedToolCallID,
						ToolName:         toolName,
						Input:            arguments,
						ProviderExecuted: boolPtr(true),
						Dynamic:          boolPtr(true),
					}

					mcpResult := map[string]any{
						"type":        "call",
						"serverLabel": serverLabel,
						"name":        name,
						"arguments":   arguments,
					}
					if output, ok := value.Item["output"]; ok && output != nil {
						mcpResult["output"] = output
					}
					if errVal, ok := value.Item["error"]; ok && errVal != nil {
						mcpResult["error"] = errVal
					}
					outCh <- languagemodel.ToolResult{
						ToolCallID: aliasedToolCallID,
						ToolName:   toolName,
						Result:     mcpResult,
						ProviderMetadata: shared.ProviderMetadata{
							argsResult.providerOptionsName: map[string]any{"itemId": itemID},
						},
					}

				case "mcp_list_tools":
					delete(ongoingToolCalls, value.OutputIndex)
					// skip

				case "apply_patch_call":
					toolCall := ongoingToolCalls[value.OutputIndex]
					if toolCall != nil && toolCall.applyPatch != nil {
						opType, _ := value.Item["operation"].(map[string]any)
						opTypeStr, _ := opType["type"].(string)
						if !toolCall.applyPatch.endEmitted && opTypeStr != "delete_file" {
							if !toolCall.applyPatch.hasDiff {
								diff, _ := opType["diff"].(string)
								outCh <- languagemodel.StreamPartToolInputDelta{
									ID:    toolCall.toolCallID,
									Delta: escapeJSONDelta(diff),
								}
							}
							outCh <- languagemodel.StreamPartToolInputDelta{
								ID:    toolCall.toolCallID,
								Delta: `"}}`,
							}
							outCh <- languagemodel.StreamPartToolInputEnd{ID: toolCall.toolCallID}
							toolCall.applyPatch.endEmitted = true
						}

						status, _ := value.Item["status"].(string)
						if status == "completed" {
							callID, _ := value.Item["call_id"].(string)
							operation := value.Item["operation"]
							itemID, _ := value.Item["id"].(string)
							outCh <- languagemodel.ToolCall{
								ToolCallID: toolCall.toolCallID,
								ToolName:   argsResult.toolNameMapping.ToCustomToolName("apply_patch"),
								Input:      jsonString(map[string]any{"callId": callID, "operation": operation}),
								ProviderMetadata: shared.ProviderMetadata{
									argsResult.providerOptionsName: map[string]any{"itemId": itemID},
								},
							}
						}
					}
					delete(ongoingToolCalls, value.OutputIndex)

				case "mcp_approval_request":
					delete(ongoingToolCalls, value.OutputIndex)
					dummyToolCallID := genID()
					itemID, _ := value.Item["id"].(string)
					approvalRequestID, _ := value.Item["approval_request_id"].(string)
					if approvalRequestID == "" {
						approvalRequestID = itemID
					}
					approvalRequestIDToDummyToolCallIDFromStream[approvalRequestID] = dummyToolCallID

					name, _ := value.Item["name"].(string)
					arguments, _ := value.Item["arguments"].(string)
					toolName := fmt.Sprintf("mcp.%s", name)

					outCh <- languagemodel.ToolCall{
						ToolCallID:       dummyToolCallID,
						ToolName:         toolName,
						Input:            arguments,
						ProviderExecuted: boolPtr(true),
						Dynamic:          boolPtr(true),
					}
					outCh <- languagemodel.ToolApprovalRequest{
						ApprovalID: approvalRequestID,
						ToolCallID: dummyToolCallID,
					}

				case "local_shell_call":
					delete(ongoingToolCalls, value.OutputIndex)
					callID, _ := value.Item["call_id"].(string)
					itemID, _ := value.Item["id"].(string)
					action, _ := value.Item["action"].(map[string]any)

					localAction := map[string]any{"type": "exec"}
					if cmd, ok := action["command"]; ok {
						localAction["command"] = cmd
					}
					if t, ok := action["timeout_ms"]; ok {
						localAction["timeoutMs"] = t
					}
					if u, ok := action["user"]; ok {
						localAction["user"] = u
					}
					if wd, ok := action["working_directory"]; ok {
						localAction["workingDirectory"] = wd
					}
					if env, ok := action["env"]; ok {
						localAction["env"] = env
					}

					outCh <- languagemodel.ToolCall{
						ToolCallID: callID,
						ToolName:   argsResult.toolNameMapping.ToCustomToolName("local_shell"),
						Input:      jsonString(map[string]any{"action": localAction}),
						ProviderMetadata: shared.ProviderMetadata{
							argsResult.providerOptionsName: map[string]any{"itemId": itemID},
						},
					}

				case "shell_call":
					delete(ongoingToolCalls, value.OutputIndex)
					callID, _ := value.Item["call_id"].(string)
					itemID, _ := value.Item["id"].(string)
					action, _ := value.Item["action"].(map[string]any)
					commands := action["commands"]

					tc := languagemodel.ToolCall{
						ToolCallID: callID,
						ToolName:   argsResult.toolNameMapping.ToCustomToolName("shell"),
						Input:      jsonString(map[string]any{"action": map[string]any{"commands": commands}}),
						ProviderMetadata: shared.ProviderMetadata{
							argsResult.providerOptionsName: map[string]any{"itemId": itemID},
						},
					}
					if argsResult.isShellProviderExecuted {
						tc.ProviderExecuted = boolPtr(true)
					}
					outCh <- tc

				case "shell_call_output":
					callID, _ := value.Item["call_id"].(string)
					outputItems := getAnySlice(value.Item, "output")
					outCh <- languagemodel.ToolResult{
						ToolCallID: callID,
						ToolName:   argsResult.toolNameMapping.ToCustomToolName("shell"),
						Result:     map[string]any{"output": convertShellOutputItems(outputItems)},
					}

				case "reasoning":
					itemID, _ := value.Item["id"].(string)
					arPart := activeReasoning[itemID]
					if arPart != nil {
						encContent, _ := value.Item["encrypted_content"].(string)
						for summaryIndex, status := range arPart.summaryParts {
							if status == "active" || status == "can-conclude" {
								outCh <- languagemodel.StreamPartReasoningEnd{
									ID: fmt.Sprintf("%s:%s", itemID, summaryIndex),
									ProviderMetadata: shared.ProviderMetadata{
										argsResult.providerOptionsName: map[string]any{
											"itemId":                    itemID,
											"reasoningEncryptedContent": nilIfEmpty(encContent),
										},
									},
								}
							}
						}
						delete(activeReasoning, itemID)
					}
				}

			case "response.function_call_arguments.delta":
				toolCall := ongoingToolCalls[value.OutputIndex]
				if toolCall != nil {
					outCh <- languagemodel.StreamPartToolInputDelta{
						ID:    toolCall.toolCallID,
						Delta: value.Delta,
					}
				}

			case "response.custom_tool_call_input.delta":
				toolCall := ongoingToolCalls[value.OutputIndex]
				if toolCall != nil {
					outCh <- languagemodel.StreamPartToolInputDelta{
						ID:    toolCall.toolCallID,
						Delta: value.Delta,
					}
				}

			case "response.apply_patch_call_operation_diff.delta":
				toolCall := ongoingToolCalls[value.OutputIndex]
				if toolCall != nil && toolCall.applyPatch != nil {
					outCh <- languagemodel.StreamPartToolInputDelta{
						ID:    toolCall.toolCallID,
						Delta: escapeJSONDelta(value.Delta),
					}
					toolCall.applyPatch.hasDiff = true
				}

			case "response.apply_patch_call_operation_diff.done":
				toolCall := ongoingToolCalls[value.OutputIndex]
				if toolCall != nil && toolCall.applyPatch != nil && !toolCall.applyPatch.endEmitted {
					if !toolCall.applyPatch.hasDiff {
						outCh <- languagemodel.StreamPartToolInputDelta{
							ID:    toolCall.toolCallID,
							Delta: escapeJSONDelta(value.Diff),
						}
						toolCall.applyPatch.hasDiff = true
					}
					outCh <- languagemodel.StreamPartToolInputDelta{
						ID:    toolCall.toolCallID,
						Delta: `"}}`,
					}
					outCh <- languagemodel.StreamPartToolInputEnd{ID: toolCall.toolCallID}
					toolCall.applyPatch.endEmitted = true
				}

			case "response.image_generation_call.partial_image":
				outCh <- languagemodel.ToolResult{
					ToolCallID:  value.ItemID,
					ToolName:    argsResult.toolNameMapping.ToCustomToolName("image_generation"),
					Result:      map[string]any{"result": value.PartialImageB64},
					Preliminary: boolPtr(true),
				}

			case "response.code_interpreter_call_code.delta":
				toolCall := ongoingToolCalls[value.OutputIndex]
				if toolCall != nil {
					outCh <- languagemodel.StreamPartToolInputDelta{
						ID:    toolCall.toolCallID,
						Delta: escapeJSONDelta(value.Delta),
					}
				}

			case "response.code_interpreter_call_code.done":
				toolCall := ongoingToolCalls[value.OutputIndex]
				if toolCall != nil {
					outCh <- languagemodel.StreamPartToolInputDelta{
						ID:    toolCall.toolCallID,
						Delta: `"}`,
					}
					outCh <- languagemodel.StreamPartToolInputEnd{ID: toolCall.toolCallID}
					outCh <- languagemodel.ToolCall{
						ToolCallID:       toolCall.toolCallID,
						ToolName:         argsResult.toolNameMapping.ToCustomToolName("code_interpreter"),
						Input:            jsonString(map[string]any{"code": value.Code, "containerId": toolCall.codeInterpreter.containerID}),
						ProviderExecuted: boolPtr(true),
					}
				}

			case "response.created":
				if value.Response != nil {
					responseID = value.Response.ID
					ts := time.Unix(value.Response.CreatedAt, 0)
					outCh <- languagemodel.StreamPartResponseMetadata{
						ResponseMetadata: languagemodel.ResponseMetadata{
							ID:        &value.Response.ID,
							Timestamp: &ts,
							ModelID:   &value.Response.Model,
						},
					}
				}

			case "response.output_text.delta":
				outCh <- languagemodel.StreamPartTextDelta{
					ID:    value.ItemID,
					Delta: value.Delta,
				}
				if opts.ProviderOptions != nil {
					if provOpts, ok := opts.ProviderOptions[argsResult.providerOptionsName]; ok {
						if provOpts["logprobs"] != nil && len(value.Logprobs) > 0 {
							logprobs = append(logprobs, value.Logprobs)
						}
					}
				}

			case "response.reasoning_summary_part.added":
				if value.SummaryIndex > 0 {
					arPart := activeReasoning[value.ItemID]
					if arPart != nil {
						arPart.summaryParts[fmt.Sprintf("%d", value.SummaryIndex)] = "active"

						// conclude can-conclude parts
						for si, status := range arPart.summaryParts {
							if status == "can-conclude" {
								outCh <- languagemodel.StreamPartReasoningEnd{
									ID: fmt.Sprintf("%s:%s", value.ItemID, si),
									ProviderMetadata: shared.ProviderMetadata{
										argsResult.providerOptionsName: map[string]any{
											"itemId": value.ItemID,
										},
									},
								}
								arPart.summaryParts[si] = "concluded"
							}
						}

						var encContent any
						if arPart.encryptedContent != nil {
							encContent = *arPart.encryptedContent
						}
						outCh <- languagemodel.StreamPartReasoningStart{
							ID: fmt.Sprintf("%s:%d", value.ItemID, value.SummaryIndex),
							ProviderMetadata: shared.ProviderMetadata{
								argsResult.providerOptionsName: map[string]any{
									"itemId":                    value.ItemID,
									"reasoningEncryptedContent": encContent,
								},
							},
						}
					}
				}

			case "response.reasoning_summary_text.delta":
				outCh <- languagemodel.StreamPartReasoningDelta{
					ID:    fmt.Sprintf("%s:%d", value.ItemID, value.SummaryIndex),
					Delta: value.Delta,
					ProviderMetadata: shared.ProviderMetadata{
						argsResult.providerOptionsName: map[string]any{
							"itemId": value.ItemID,
						},
					},
				}

			case "response.reasoning_summary_part.done":
				arPart := activeReasoning[value.ItemID]
				if arPart != nil {
					siKey := fmt.Sprintf("%d", value.SummaryIndex)
					if argsResult.store != nil && *argsResult.store {
						outCh <- languagemodel.StreamPartReasoningEnd{
							ID: fmt.Sprintf("%s:%d", value.ItemID, value.SummaryIndex),
							ProviderMetadata: shared.ProviderMetadata{
								argsResult.providerOptionsName: map[string]any{
									"itemId": value.ItemID,
								},
							},
						}
						arPart.summaryParts[siKey] = "concluded"
					} else {
						arPart.summaryParts[siKey] = "can-conclude"
					}
				}

			case "response.completed", "response.incomplete":
				if value.Response != nil {
					var reason *string
					if value.Response.IncompleteDetails != nil {
						reason = &value.Response.IncompleteDetails.Reason
					}
					finishReason = languagemodel.FinishReason{
						Unified: MapOpenAIResponseFinishReason(MapOpenAIResponseFinishReasonOptions{
							FinishReason:    reason,
							HasFunctionCall: hasFunctionCall,
						}),
						Raw: reason,
					}
					usage = value.Response.Usage
					if value.Response.ServiceTier != nil {
						serviceTier = *value.Response.ServiceTier
					}
				}

			case "response.output_text.annotation.added":
				if value.Annotation != nil {
					ongoingAnnotations = append(ongoingAnnotations, value.Annotation)
					sources := m.createSourceFromAnnotation(value.Annotation, argsResult.providerOptionsName, genID)
					for _, s := range sources {
						if sp, ok := s.(languagemodel.StreamPart); ok {
							outCh <- sp
						}
					}
				}

			case "error":
				outCh <- languagemodel.StreamPartError{Error: value}
			}
		}

		// flush
		providerMetadata := shared.ProviderMetadata{
			argsResult.providerOptionsName: map[string]any{
				"responseId": responseID,
			},
		}
		if len(logprobs) > 0 {
			providerMetadata[argsResult.providerOptionsName]["logprobs"] = logprobs
		}
		if serviceTier != "" {
			providerMetadata[argsResult.providerOptionsName]["serviceTier"] = serviceTier
		}

		outCh <- languagemodel.StreamPartFinish{
			FinishReason:     finishReason,
			Usage:            ConvertOpenAIResponsesUsage(usage),
			ProviderMetadata: providerMetadata,
		}
	}()

	return languagemodel.StreamResult{
		Stream:   outCh,
		Request:  &languagemodel.StreamResultRequest{Body: body},
		Response: &languagemodel.StreamResultResponse{Headers: responseHeaders},
	}, nil
}

// createSourceFromAnnotation converts an annotation map to source Content items.
func (m *OpenAIResponsesLanguageModel) createSourceFromAnnotation(annotation map[string]any, providerOptionsName string, genID func() string) []languagemodel.Content {
	annType, _ := annotation["type"].(string)

	switch annType {
	case "url_citation":
		url, _ := annotation["url"].(string)
		title, _ := annotation["title"].(string)
		return []languagemodel.Content{
			languagemodel.SourceURL{
				ID:    genID(),
				URL:   url,
				Title: &title,
			},
		}
	case "file_citation":
		fileID, _ := annotation["file_id"].(string)
		filename, _ := annotation["filename"].(string)
		index, _ := annotation["index"].(float64)
		return []languagemodel.Content{
			languagemodel.SourceDocument{
				ID:        genID(),
				MediaType: "text/plain",
				Title:     filename,
				Filename:  &filename,
				ProviderMetadata: shared.ProviderMetadata{
					providerOptionsName: map[string]any{
						"type":   "file_citation",
						"fileId": fileID,
						"index":  int(index),
					},
				},
			},
		}
	case "container_file_citation":
		fileID, _ := annotation["file_id"].(string)
		filename, _ := annotation["filename"].(string)
		containerID, _ := annotation["container_id"].(string)
		return []languagemodel.Content{
			languagemodel.SourceDocument{
				ID:        genID(),
				MediaType: "text/plain",
				Title:     filename,
				Filename:  &filename,
				ProviderMetadata: shared.ProviderMetadata{
					providerOptionsName: map[string]any{
						"type":        "container_file_citation",
						"fileId":      fileID,
						"containerId": containerID,
					},
				},
			},
		}
	case "file_path":
		fileID, _ := annotation["file_id"].(string)
		index, _ := annotation["index"].(float64)
		return []languagemodel.Content{
			languagemodel.SourceDocument{
				ID:        genID(),
				MediaType: "application/octet-stream",
				Title:     fileID,
				Filename:  &fileID,
				ProviderMetadata: shared.ProviderMetadata{
					providerOptionsName: map[string]any{
						"type":   "file_path",
						"fileId": fileID,
						"index":  int(index),
					},
				},
			},
		}
	}
	return nil
}

// generateID returns the ID generator function, using config or default.
func (m *OpenAIResponsesLanguageModel) generateID() func() string {
	if m.config.GenerateID != nil {
		return m.config.GenerateID
	}
	return providerutils.GenerateId
}

// mapWebSearchOutput converts a web search action to the output format.
func mapWebSearchOutput(action map[string]any) map[string]any {
	if action == nil {
		return map[string]any{}
	}

	actionType, _ := action["type"].(string)
	switch actionType {
	case "search":
		result := map[string]any{
			"action": map[string]any{"type": "search"},
		}
		if query, ok := action["query"]; ok {
			result["action"].(map[string]any)["query"] = query
		}
		if sources, ok := action["sources"]; ok && sources != nil {
			result["sources"] = sources
		}
		return result
	case "open_page":
		return map[string]any{
			"action": map[string]any{"type": "openPage", "url": action["url"]},
		}
	case "find_in_page":
		return map[string]any{
			"action": map[string]any{
				"type":    "findInPage",
				"url":     action["url"],
				"pattern": action["pattern"],
			},
		}
	}
	return map[string]any{}
}

// escapeJSONDelta escapes a string for embedding inside a JSON string.
// Uses JSON.stringify and removes the outer quotes.
func escapeJSONDelta(delta string) string {
	b, err := json.Marshal(delta)
	if err != nil {
		return delta
	}
	// Remove outer quotes
	s := string(b)
	if len(s) >= 2 {
		return s[1 : len(s)-1]
	}
	return delta
}

// boolPtr returns a pointer to the given bool value.
func boolPtr(v bool) *bool {
	return &v
}

// jsonString marshals a value to a JSON string.
func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// setIfNotNil sets a key in a map only if the value pointer is not nil.
func setIfNotNil[T any](m map[string]any, key string, v *T) {
	if v != nil {
		m[key] = *v
	}
}

// nilIfEmpty returns nil if the string is empty, otherwise the string.
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nilIfEmptyPtr returns nil if the pointer is nil, otherwise the dereferenced value.
func nilIfEmptyPtr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// getMapSlice extracts a []map[string]any from a map field.
func getMapSlice(m map[string]any, key string) []map[string]any {
	val, ok := m[key]
	if !ok || val == nil {
		return nil
	}
	if arr, ok := val.([]any); ok {
		result := make([]map[string]any, 0, len(arr))
		for _, item := range arr {
			if itemMap, ok := item.(map[string]any); ok {
				result = append(result, itemMap)
			}
		}
		return result
	}
	if arr, ok := val.([]map[string]any); ok {
		return arr
	}
	return nil
}

// getAnySlice extracts a []any from a map field.
func getAnySlice(m map[string]any, key string) []any {
	val, ok := m[key]
	if !ok || val == nil {
		return nil
	}
	if arr, ok := val.([]any); ok {
		return arr
	}
	return nil
}

// convertLogprobs converts a []any to []OpenAIResponsesLogprob.
func convertLogprobs(lp []any) []OpenAIResponsesLogprob {
	var result []OpenAIResponsesLogprob
	for _, item := range lp {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		token, _ := itemMap["token"].(string)
		logprob, _ := itemMap["logprob"].(float64)
		var topLogprobs []OpenAIResponsesTopLogprobItem
		if tlp, ok := itemMap["top_logprobs"].([]any); ok {
			for _, tlpItem := range tlp {
				tlpMap, ok := tlpItem.(map[string]any)
				if !ok {
					continue
				}
				topToken, _ := tlpMap["token"].(string)
				topLogprob, _ := tlpMap["logprob"].(float64)
				topLogprobs = append(topLogprobs, OpenAIResponsesTopLogprobItem{
					Token:   topToken,
					Logprob: topLogprob,
				})
			}
		}
		result = append(result, OpenAIResponsesLogprob{
			Token:       token,
			Logprob:     logprob,
			TopLogprobs: topLogprobs,
		})
	}
	return result
}

// convertShellOutputItems converts shell output items from API format.
func convertShellOutputItems(items []any) []map[string]any {
	var result []map[string]any
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		stdout, _ := itemMap["stdout"].(string)
		stderr, _ := itemMap["stderr"].(string)
		outcome, _ := itemMap["outcome"].(map[string]any)
		outcomeType, _ := outcome["type"].(string)

		entry := map[string]any{
			"stdout": stdout,
			"stderr": stderr,
		}
		if outcomeType == "exit" {
			exitCode, _ := outcome["exit_code"].(float64)
			entry["outcome"] = map[string]any{"type": "exit", "exitCode": int(exitCode)}
		} else {
			entry["outcome"] = map[string]any{"type": "timeout"}
		}
		result = append(result, entry)
	}
	return result
}

// convertFileSearchResults converts file_search results from the API format.
func convertFileSearchResults(part map[string]any) any {
	results, ok := part["results"].([]any)
	if !ok || results == nil {
		return nil
	}
	var converted []map[string]any
	for _, r := range results {
		rMap, ok := r.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{
			"attributes": rMap["attributes"],
			"fileId":     rMap["file_id"],
			"filename":   rMap["filename"],
			"score":      rMap["score"],
			"text":       rMap["text"],
		}
		converted = append(converted, entry)
	}
	return converted
}

// parseResponsesProviderOptions manually parses provider options for the responses model.
// This avoids needing a Zod-like schema in Go; we just extract fields from the map.
func parseResponsesProviderOptions(provider string, providerOptions shared.ProviderOptions) *OpenAILanguageModelResponsesOptions {
	if providerOptions == nil {
		return nil
	}
	optsMap, ok := providerOptions[provider]
	if !ok || optsMap == nil {
		return nil
	}

	result := &OpenAILanguageModelResponsesOptions{}
	hasFields := false

	if v, ok := optsMap["conversation"].(string); ok {
		result.Conversation = &v
		hasFields = true
	}
	if v, ok := optsMap["include"].([]any); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				result.Include = append(result.Include, s)
			}
		}
		hasFields = true
	}
	if v, ok := optsMap["instructions"].(string); ok {
		result.Instructions = &v
		hasFields = true
	}
	if v, ok := optsMap["logprobs"]; ok {
		switch lp := v.(type) {
		case bool:
			result.Logprobs = &LogprobsSetting{BoolValue: &lp}
			hasFields = true
		case float64:
			intVal := int(lp)
			result.Logprobs = &LogprobsSetting{IntValue: &intVal}
			hasFields = true
		}
	}
	if v, ok := optsMap["maxToolCalls"].(float64); ok {
		intVal := int(v)
		result.MaxToolCalls = &intVal
		hasFields = true
	}
	if v, ok := optsMap["metadata"]; ok {
		result.Metadata = v
		hasFields = true
	}
	if v, ok := optsMap["parallelToolCalls"].(bool); ok {
		result.ParallelToolCalls = &v
		hasFields = true
	}
	if v, ok := optsMap["previousResponseId"].(string); ok {
		result.PreviousResponseID = &v
		hasFields = true
	}
	if v, ok := optsMap["promptCacheKey"].(string); ok {
		result.PromptCacheKey = &v
		hasFields = true
	}
	if v, ok := optsMap["promptCacheRetention"].(string); ok {
		result.PromptCacheRetention = &v
		hasFields = true
	}
	if v, ok := optsMap["reasoningEffort"].(string); ok {
		result.ReasoningEffort = &v
		hasFields = true
	}
	if v, ok := optsMap["reasoningSummary"].(string); ok {
		result.ReasoningSummary = &v
		hasFields = true
	}
	if v, ok := optsMap["safetyIdentifier"].(string); ok {
		result.SafetyIdentifier = &v
		hasFields = true
	}
	if v, ok := optsMap["serviceTier"].(string); ok {
		result.ServiceTier = &v
		hasFields = true
	}
	if v, ok := optsMap["store"].(bool); ok {
		result.Store = &v
		hasFields = true
	}
	if v, ok := optsMap["strictJsonSchema"].(bool); ok {
		result.StrictJSONSchema = &v
		hasFields = true
	}
	if v, ok := optsMap["textVerbosity"].(string); ok {
		result.TextVerbosity = &v
		hasFields = true
	}
	if v, ok := optsMap["truncation"].(string); ok {
		result.Truncation = &v
		hasFields = true
	}
	if v, ok := optsMap["user"].(string); ok {
		result.User = &v
		hasFields = true
	}
	if v, ok := optsMap["systemMessageMode"].(string); ok {
		result.SystemMessageMode = &v
		hasFields = true
	}
	if v, ok := optsMap["forceReasoning"].(bool); ok {
		result.ForceReasoning = &v
		hasFields = true
	}

	if !hasFields {
		return nil
	}
	return result
}

// openaiResponsesResponseSchema is the schema for parsing non-streaming Responses API responses.
var openaiResponsesResponseSchema = &providerutils.Schema[OpenAIResponsesResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[OpenAIResponsesResponse], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[OpenAIResponsesResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		var response OpenAIResponsesResponse
		if err := json.Unmarshal(data, &response); err != nil {
			return &providerutils.ValidationResult[OpenAIResponsesResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[OpenAIResponsesResponse]{
			Success: true,
			Value:   response,
		}, nil
	},
}

// openaiResponsesChunkSchema is the schema for parsing streaming chunks.
var openaiResponsesChunkSchema = &providerutils.Schema[OpenAIResponsesChunk]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[OpenAIResponsesChunk], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[OpenAIResponsesChunk]{
				Success: false,
				Error:   err,
			}, nil
		}
		var chunk OpenAIResponsesChunk
		if err := json.Unmarshal(data, &chunk); err != nil {
			return &providerutils.ValidationResult[OpenAIResponsesChunk]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[OpenAIResponsesChunk]{
			Success: true,
			Value:   chunk,
		}, nil
	},
}

// Ensure interface compliance.
var _ languagemodel.LanguageModel = (*OpenAIResponsesLanguageModel)(nil)
