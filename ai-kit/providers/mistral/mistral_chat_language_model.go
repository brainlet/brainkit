// Ported from: packages/mistral/src/mistral-chat-language-model.ts
package mistral

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ChatConfig holds the configuration for a Mistral chat language model.
type ChatConfig struct {
	// Provider is the provider identifier (e.g. "mistral.chat").
	Provider string

	// BaseURL is the base URL for the Mistral API.
	BaseURL string

	// Headers returns the HTTP headers to include with requests.
	Headers func() map[string]string

	// Fetch is an optional custom HTTP fetch function.
	Fetch providerutils.FetchFunction

	// GenerateID is an optional custom ID generator.
	GenerateID providerutils.IdGenerator
}

// ChatLanguageModel implements languagemodel.LanguageModel for the Mistral API.
type ChatLanguageModel struct {
	modelID    MistralChatModelId
	config     ChatConfig
	generateID providerutils.IdGenerator
}

// NewChatLanguageModel creates a new ChatLanguageModel.
func NewChatLanguageModel(modelID MistralChatModelId, config ChatConfig) *ChatLanguageModel {
	genID := config.GenerateID
	if genID == nil {
		genID = providerutils.GenerateId
	}

	return &ChatLanguageModel{
		modelID:    modelID,
		config:     config,
		generateID: genID,
	}
}

// SpecificationVersion returns "v3".
func (m *ChatLanguageModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *ChatLanguageModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *ChatLanguageModel) ModelID() string { return m.modelID }

// SupportedUrls returns supported URL patterns. Mistral supports PDF URLs.
func (m *ChatLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return map[string][]*regexp.Regexp{
		"application/pdf": {regexp.MustCompile(`^https://.*$`)},
	}, nil
}

// getArgs prepares the request arguments from CallOptions.
func (m *ChatLanguageModel) getArgs(opts languagemodel.CallOptions) (args map[string]any, warnings []shared.Warning, err error) {
	warnings = []shared.Warning{}

	// Convert ProviderOptions to map[string]interface{} for ParseProviderOptions
	providerOptsMap := toInterfaceMap(opts.ProviderOptions)

	// Parse provider options
	parsedOptions, err := providerutils.ParseProviderOptions(
		"mistral",
		providerOptsMap,
		MistralLanguageModelOptionsSchema,
	)
	if err != nil {
		return nil, nil, err
	}

	options := &MistralLanguageModelOptions{}
	if parsedOptions != nil {
		options = parsedOptions
	}

	if opts.TopK != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "topK"})
	}

	if opts.FrequencyPenalty != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "frequencyPenalty"})
	}

	if opts.PresencePenalty != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "presencePenalty"})
	}

	if len(opts.StopSequences) > 0 {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "stopSequences"})
	}

	structuredOutputs := true
	if options.StructuredOutputs != nil {
		structuredOutputs = *options.StructuredOutputs
	}

	strictJSONSchema := false
	if options.StrictJSONSchema != nil {
		strictJSONSchema = *options.StrictJSONSchema
	}

	prompt := opts.Prompt

	// For Mistral we need to instruct the model to return a JSON object
	// when no schema is provided.
	if jsonFormat, ok := opts.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
		if jsonFormat.Schema == nil {
			// Inject JSON instruction into messages
			prompt = injectJsonInstructionIntoMessages(prompt, jsonFormat.Schema)
		}
	}

	// Build base args
	args = map[string]any{
		"model": m.modelID,
	}

	if options.SafePrompt != nil {
		args["safe_prompt"] = *options.SafePrompt
	}

	if opts.MaxOutputTokens != nil {
		args["max_tokens"] = *opts.MaxOutputTokens
	}
	if opts.Temperature != nil {
		args["temperature"] = *opts.Temperature
	}
	if opts.TopP != nil {
		args["top_p"] = *opts.TopP
	}
	if opts.Seed != nil {
		args["random_seed"] = *opts.Seed
	}

	// Response format
	if jsonFormat, ok := opts.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
		if structuredOutputs && jsonFormat.Schema != nil {
			name := "response"
			if jsonFormat.Name != nil {
				name = *jsonFormat.Name
			}
			jsonSchemaObj := map[string]any{
				"schema": jsonFormat.Schema,
				"strict": strictJSONSchema,
				"name":   name,
			}
			if jsonFormat.Description != nil {
				jsonSchemaObj["description"] = *jsonFormat.Description
			}
			args["response_format"] = map[string]any{
				"type":        "json_schema",
				"json_schema": jsonSchemaObj,
			}
		} else {
			args["response_format"] = map[string]any{
				"type": "json_object",
			}
		}
	}

	// Mistral-specific provider options
	if options.DocumentImageLimit != nil {
		args["document_image_limit"] = *options.DocumentImageLimit
	}
	if options.DocumentPageLimit != nil {
		args["document_page_limit"] = *options.DocumentPageLimit
	}

	// Messages
	args["messages"] = ConvertToMistralChatMessages(prompt)

	// Prepare tools
	toolResult := PrepareTools(opts.Tools, opts.ToolChoice)

	if toolResult.Tools != nil {
		args["tools"] = toolResult.Tools
	}
	if toolResult.ToolChoice != nil {
		args["tool_choice"] = toolResult.ToolChoice
	}

	// Parallel tool calls
	if toolResult.Tools != nil && options.ParallelToolCalls != nil {
		args["parallel_tool_calls"] = *options.ParallelToolCalls
	}

	warnings = append(warnings, toolResult.ToolWarnings...)

	return args, warnings, nil
}

// DoGenerate implements languagemodel.LanguageModel.DoGenerate for non-streaming.
func (m *ChatLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	args, warnings, err := m.getArgs(options)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[mistralChatResponse]{
		URL:                       fmt.Sprintf("%s/chat/completions", m.config.BaseURL),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers)),
		Body:                      args,
		FailedResponseHandler:     mistralFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(mistralChatResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	response := result.Value
	choice := response.Choices[0]
	var content []languagemodel.Content

	// Process content parts in order to preserve sequence
	if choice.Message.ContentParts != nil {
		for _, part := range choice.Message.ContentParts {
			switch part.Type {
			case "thinking":
				reasoningText := extractReasoningContent(part.Thinking)
				if len(reasoningText) > 0 {
					content = append(content, languagemodel.Reasoning{Text: reasoningText})
				}
			case "text":
				if len(part.Text) > 0 {
					content = append(content, languagemodel.Text{Text: part.Text})
				}
			}
		}
	} else if choice.Message.Content != nil && len(*choice.Message.Content) > 0 {
		// Handle legacy string content
		content = append(content, languagemodel.Text{Text: *choice.Message.Content})
	}

	// Tool calls
	if choice.Message.ToolCalls != nil {
		for _, toolCall := range choice.Message.ToolCalls {
			content = append(content, languagemodel.ToolCall{
				ToolCallID: toolCall.ID,
				ToolName:   toolCall.Function.Name,
				Input:      toolCall.Function.Arguments,
			})
		}
	}

	// Finish reason
	finishReason := languagemodel.FinishReason{
		Unified: MapMistralFinishReason(choice.FinishReason),
	}
	if choice.FinishReason != nil {
		finishReason.Raw = choice.FinishReason
	}

	// Response metadata
	respMeta := getResponseMetadata(response.ID, response.Model, response.Created)

	return languagemodel.GenerateResult{
		Content:      content,
		FinishReason: finishReason,
		Usage:        ConvertMistralUsage(response.Usage),
		Request:      &languagemodel.GenerateResultRequest{Body: args},
		Response: &languagemodel.GenerateResultResponse{
			ResponseMetadata: respMeta,
			Headers:          result.ResponseHeaders,
			Body:             result.RawValue,
		},
		Warnings: warnings,
	}, nil
}

// DoStream implements languagemodel.LanguageModel.DoStream for streaming.
func (m *ChatLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	args, warnings, err := m.getArgs(options)
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	body := make(map[string]any)
	for k, v := range args {
		body[k] = v
	}
	body["stream"] = true

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[<-chan providerutils.ParseResult[any]]{
		URL:                       fmt.Sprintf("%s/chat/completions", m.config.BaseURL),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     mistralFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(mistralChatChunkSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	inputChan := result.Value
	outputChan := make(chan languagemodel.StreamPart)

	includeRawChunks := options.IncludeRawChunks != nil && *options.IncludeRawChunks
	generateID := m.generateID

	go func() {
		defer close(outputChan)

		finishReason := languagemodel.FinishReason{
			Unified: languagemodel.FinishReasonOther,
		}
		var usage *MistralUsage
		isFirstChunk := true
		activeText := false
		var activeReasoningID *string

		// Start event
		outputChan <- languagemodel.StreamPartStreamStart{Warnings: warnings}

		for chunk := range inputChan {
			// Emit raw chunk if requested
			if includeRawChunks {
				outputChan <- languagemodel.StreamPartRaw{RawValue: chunk.RawValue}
			}

			if !chunk.Success {
				outputChan <- languagemodel.StreamPartError{Error: chunk.Error}
				continue
			}

			chunkMap, ok := chunk.Value.(map[string]any)
			if !ok {
				continue
			}

			if isFirstChunk {
				isFirstChunk = false
				respMeta := getResponseMetadataFromMap(chunkMap)
				outputChan <- languagemodel.StreamPartResponseMetadata{
					ResponseMetadata: respMeta,
				}
			}

			// Handle usage
			if usageRaw, ok := chunkMap["usage"]; ok && usageRaw != nil {
				usage = parseMistralUsage(usageRaw)
			}

			// Handle choices
			choicesRaw, ok := chunkMap["choices"]
			if !ok {
				continue
			}
			choices, ok := choicesRaw.([]any)
			if !ok || len(choices) == 0 {
				continue
			}
			choiceMap, ok := choices[0].(map[string]any)
			if !ok {
				continue
			}

			// Check delta
			deltaRaw, ok := choiceMap["delta"]
			if !ok || deltaRaw == nil {
				// Check finish_reason even without delta
				if fr, ok := choiceMap["finish_reason"]; ok && fr != nil {
					frStr := fmt.Sprintf("%v", fr)
					finishReason = languagemodel.FinishReason{
						Unified: MapMistralFinishReason(&frStr),
						Raw:     &frStr,
					}
				}
				continue
			}
			delta, ok := deltaRaw.(map[string]any)
			if !ok {
				continue
			}

			// Process array content for thinking/reasoning
			if contentRaw, ok := delta["content"]; ok && contentRaw != nil {
				if contentArr, ok := contentRaw.([]any); ok {
					for _, partRaw := range contentArr {
						partMap, ok := partRaw.(map[string]any)
						if !ok {
							continue
						}
						partType, _ := partMap["type"].(string)
						if partType == "thinking" {
							thinkingRaw, _ := partMap["thinking"].([]any)
							reasoningDelta := extractReasoningContentFromRaw(thinkingRaw)
							if len(reasoningDelta) > 0 {
								if activeReasoningID == nil {
									// End any active text before starting reasoning
									if activeText {
										outputChan <- languagemodel.StreamPartTextEnd{ID: "0"}
										activeText = false
									}
									id := generateID()
									activeReasoningID = &id
									outputChan <- languagemodel.StreamPartReasoningStart{ID: *activeReasoningID}
								}
								outputChan <- languagemodel.StreamPartReasoningDelta{
									ID:    *activeReasoningID,
									Delta: reasoningDelta,
								}
							}
						}
					}
				}
			}

			// Extract text content
			textContent := extractTextContentFromRaw(delta["content"])
			if textContent != nil && len(*textContent) > 0 {
				if !activeText {
					// If we were in reasoning mode, end it before starting text
					if activeReasoningID != nil {
						outputChan <- languagemodel.StreamPartReasoningEnd{ID: *activeReasoningID}
						activeReasoningID = nil
					}
					outputChan <- languagemodel.StreamPartTextStart{ID: "0"}
					activeText = true
				}
				outputChan <- languagemodel.StreamPartTextDelta{
					ID:    "0",
					Delta: *textContent,
				}
			}

			// Tool calls
			if toolCallsRaw, ok := delta["tool_calls"]; ok && toolCallsRaw != nil {
				toolCallDeltas, ok := toolCallsRaw.([]any)
				if ok {
					for _, tcRaw := range toolCallDeltas {
						tcMap, ok := tcRaw.(map[string]any)
						if !ok {
							continue
						}

						tcID, _ := tcMap["id"].(string)
						funcName := ""
						funcArgs := ""
						if fnObj, ok := tcMap["function"].(map[string]any); ok {
							funcName, _ = fnObj["name"].(string)
							funcArgs, _ = fnObj["arguments"].(string)
						}

						outputChan <- languagemodel.StreamPartToolInputStart{
							ID:       tcID,
							ToolName: funcName,
						}

						outputChan <- languagemodel.StreamPartToolInputDelta{
							ID:    tcID,
							Delta: funcArgs,
						}

						outputChan <- languagemodel.StreamPartToolInputEnd{ID: tcID}

						outputChan <- languagemodel.ToolCall{
							ToolCallID: tcID,
							ToolName:   funcName,
							Input:      funcArgs,
						}
					}
				}
			}

			// Check finish_reason
			if fr, ok := choiceMap["finish_reason"]; ok && fr != nil {
				frStr := fmt.Sprintf("%v", fr)
				finishReason = languagemodel.FinishReason{
					Unified: MapMistralFinishReason(&frStr),
					Raw:     &frStr,
				}
			}
		}

		// Flush: end active blocks
		if activeReasoningID != nil {
			outputChan <- languagemodel.StreamPartReasoningEnd{ID: *activeReasoningID}
		}
		if activeText {
			outputChan <- languagemodel.StreamPartTextEnd{ID: "0"}
		}

		// Finish event
		outputChan <- languagemodel.StreamPartFinish{
			FinishReason: finishReason,
			Usage:        ConvertMistralUsage(usage),
		}
	}()

	return languagemodel.StreamResult{
		Stream:   outputChan,
		Request:  &languagemodel.StreamResultRequest{Body: body},
		Response: &languagemodel.StreamResultResponse{Headers: result.ResponseHeaders},
	}, nil
}

// --- Helper functions ---

// extractReasoningContent extracts reasoning text from thinking parts.
func extractReasoningContent(thinking []mistralThinkingPart) string {
	var parts []string
	for _, chunk := range thinking {
		if chunk.Type == "text" {
			parts = append(parts, chunk.Text)
		}
	}
	return strings.Join(parts, "")
}

// extractReasoningContentFromRaw extracts reasoning text from raw thinking array.
func extractReasoningContentFromRaw(thinking []any) string {
	var parts []string
	for _, item := range thinking {
		if m, ok := item.(map[string]any); ok {
			if t, _ := m["type"].(string); t == "text" {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
	}
	return strings.Join(parts, "")
}

// extractTextContentFromRaw extracts text content from a content value.
// Content can be a string or an array of content parts.
func extractTextContentFromRaw(content any) *string {
	if content == nil {
		return nil
	}

	if s, ok := content.(string); ok {
		return &s
	}

	if arr, ok := content.([]any); ok {
		var textContent []string
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				t, _ := m["type"].(string)
				switch t {
				case "text":
					if text, ok := m["text"].(string); ok {
						textContent = append(textContent, text)
					}
				case "thinking", "image_url", "reference":
					// ignored
				}
			}
		}
		if len(textContent) > 0 {
			result := strings.Join(textContent, "")
			return &result
		}
		return nil
	}

	return nil
}

// injectJsonInstructionIntoMessages injects a JSON instruction into the last
// user message in the prompt when no schema is provided for JSON format.
func injectJsonInstructionIntoMessages(messages languagemodel.Prompt, schema map[string]any) languagemodel.Prompt {
	result := make(languagemodel.Prompt, len(messages))
	copy(result, messages)

	// Find the last user message and inject JSON instruction
	for i := len(result) - 1; i >= 0; i-- {
		if userMsg, ok := result[i].(languagemodel.UserMessage); ok {
			injectedText := providerutils.InjectJsonInstruction(providerutils.InjectJsonInstructionOptions{
				Schema: schema,
			})
			userMsg.Content = append(userMsg.Content, languagemodel.TextPart{
				Text: injectedText,
			})
			result[i] = userMsg
			break
		}
	}

	return result
}

// getResponseMetadataFromMap extracts response metadata from a streaming chunk map.
func getResponseMetadataFromMap(m map[string]any) languagemodel.ResponseMetadata {
	rm := languagemodel.ResponseMetadata{}
	if id, ok := m["id"].(string); ok {
		rm.ID = &id
	}
	if model, ok := m["model"].(string); ok {
		rm.ModelID = &model
	}
	if created, ok := m["created"].(float64); ok {
		createdF := created
		rm = getResponseMetadata(rm.ID, rm.ModelID, &createdF)
	}
	return rm
}

// parseMistralUsage parses a raw usage value into MistralUsage.
func parseMistralUsage(raw any) *MistralUsage {
	usageMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	usage := &MistralUsage{}
	if v, ok := usageMap["prompt_tokens"].(float64); ok {
		usage.PromptTokens = int(v)
	}
	if v, ok := usageMap["completion_tokens"].(float64); ok {
		usage.CompletionTokens = int(v)
	}
	if v, ok := usageMap["total_tokens"].(float64); ok {
		usage.TotalTokens = int(v)
	}
	return usage
}

// toInterfaceMap converts shared.ProviderOptions to map[string]interface{}.
func toInterfaceMap(opts shared.ProviderOptions) map[string]interface{} {
	if opts == nil {
		return nil
	}
	result := make(map[string]interface{}, len(opts))
	for k, v := range opts {
		result[k] = v
	}
	return result
}

// convertOptionalHeaders converts map[string]*string to map[string]string.
func convertOptionalHeaders(headers map[string]*string) map[string]string {
	if headers == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range headers {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}

// --- Response schemas (Go structs replacing Zod schemas) ---

// mistralThinkingPart represents a thinking part within content.
type mistralThinkingPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// mistralContentPart represents a content part in a Mistral response.
type mistralContentPart struct {
	Type     string                `json:"type"`
	Text     string                `json:"text,omitempty"`
	Thinking []mistralThinkingPart `json:"thinking,omitempty"`
}

// mistralResponseMessage represents a message in a Mistral chat completion response.
type mistralResponseMessage struct {
	Role         string                          `json:"role"`
	Content      *string                         `json:"content,omitempty"`
	ContentParts []mistralContentPart            `json:"-"` // Handled by custom unmarshal
	ToolCalls    []mistralResponseToolCall        `json:"tool_calls,omitempty"`
}

// UnmarshalJSON handles the dual content format (string or array).
func (m *mistralResponseMessage) UnmarshalJSON(data []byte) error {
	// First try to unmarshal with a raw content field
	type Alias struct {
		Role      string                    `json:"role"`
		Content   json.RawMessage           `json:"content,omitempty"`
		ToolCalls []mistralResponseToolCall  `json:"tool_calls,omitempty"`
	}

	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	m.Role = alias.Role
	m.ToolCalls = alias.ToolCalls

	if alias.Content == nil || string(alias.Content) == "null" {
		return nil
	}

	// Try string first
	var s string
	if err := json.Unmarshal(alias.Content, &s); err == nil {
		m.Content = &s
		return nil
	}

	// Try array of content parts
	var parts []mistralContentPart
	if err := json.Unmarshal(alias.Content, &parts); err == nil {
		m.ContentParts = parts
		return nil
	}

	return nil
}

// mistralResponseToolCall represents a tool call in a Mistral response.
type mistralResponseToolCall struct {
	ID       string                          `json:"id"`
	Function mistralResponseToolCallFunction `json:"function"`
}

// mistralResponseToolCallFunction represents the function part of a tool call.
type mistralResponseToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// mistralChatChoice represents a choice in a Mistral chat completion response.
type mistralChatChoice struct {
	Message      mistralResponseMessage `json:"message"`
	Index        int                    `json:"index"`
	FinishReason *string                `json:"finish_reason,omitempty"`
}

// mistralChatResponse represents a Mistral chat completion response.
type mistralChatResponse struct {
	ID      *string            `json:"id,omitempty"`
	Created *float64           `json:"created,omitempty"`
	Model   *string            `json:"model,omitempty"`
	Choices []mistralChatChoice `json:"choices"`
	Object  string             `json:"object"`
	Usage   *MistralUsage      `json:"usage,omitempty"`
}

// mistralChatResponseSchema is the schema for Mistral chat responses.
var mistralChatResponseSchema = &providerutils.Schema[mistralChatResponse]{}

// mistralChatChunkSchema is the schema for Mistral streaming chunks.
// Parses into map[string]any since the chunk structure varies.
var mistralChatChunkSchema = &providerutils.Schema[any]{}

// Verify ChatLanguageModel implements the LanguageModel interface.
var _ languagemodel.LanguageModel = (*ChatLanguageModel)(nil)
