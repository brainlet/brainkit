// Ported from: packages/huggingface/src/responses/huggingface-responses-language-model.ts
package huggingface

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ResponsesLanguageModel implements languagemodel.LanguageModel for the
// HuggingFace Responses API.
type ResponsesLanguageModel struct {
	modelID ResponsesModelID
	config  Config
}

// NewResponsesLanguageModel creates a new ResponsesLanguageModel.
func NewResponsesLanguageModel(modelID ResponsesModelID, config Config) *ResponsesLanguageModel {
	return &ResponsesLanguageModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *ResponsesLanguageModel) SpecificationVersion() string {
	return "v3"
}

// Provider returns the provider identifier.
func (m *ResponsesLanguageModel) Provider() string {
	return m.config.Provider
}

// ModelID returns the model ID.
func (m *ResponsesLanguageModel) ModelID() string {
	return m.modelID
}

// SupportedUrls returns URL patterns supported natively by the model.
func (m *ResponsesLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return map[string][]*regexp.Regexp{
		"image/*": {regexp.MustCompile(`^https?://.*$`)},
	}, nil
}

// --- Provider options schema ---

// responsesProviderOptions holds parsed HuggingFace provider options.
type responsesProviderOptions struct {
	Metadata         map[string]string `json:"metadata,omitempty"`
	Instructions     *string           `json:"instructions,omitempty"`
	StrictJSONSchema *bool             `json:"strictJsonSchema,omitempty"`
	ReasoningEffort  *string           `json:"reasoningEffort,omitempty"`
}

var responsesProviderOptionsSchema = &providerutils.Schema[responsesProviderOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[responsesProviderOptions], error) {
		m, ok := value.(map[string]interface{})
		if !ok {
			return &providerutils.ValidationResult[responsesProviderOptions]{
				Success: true,
				Value:   responsesProviderOptions{},
			}, nil
		}

		var opts responsesProviderOptions

		if metadata, ok := m["metadata"].(map[string]interface{}); ok {
			opts.Metadata = make(map[string]string)
			for k, v := range metadata {
				if s, ok := v.(string); ok {
					opts.Metadata[k] = s
				}
			}
		}

		if instructions, ok := m["instructions"].(string); ok {
			opts.Instructions = &instructions
		}

		if strict, ok := m["strictJsonSchema"].(bool); ok {
			opts.StrictJSONSchema = &strict
		}

		if effort, ok := m["reasoningEffort"].(string); ok {
			opts.ReasoningEffort = &effort
		}

		return &providerutils.ValidationResult[responsesProviderOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}

// --- getArgs ---

type getArgsResult struct {
	args     map[string]any
	warnings []shared.Warning
}

func (m *ResponsesLanguageModel) getArgs(options languagemodel.CallOptions) (getArgsResult, error) {
	warnings := []shared.Warning{}

	if options.TopK != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "topK"})
	}
	if options.Seed != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "seed"})
	}
	if options.PresencePenalty != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "presencePenalty"})
	}
	if options.FrequencyPenalty != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "frequencyPenalty"})
	}
	if options.StopSequences != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "stopSequences"})
	}

	msgResult, err := convertToHuggingFaceResponsesMessages(options.Prompt)
	if err != nil {
		return getArgsResult{}, err
	}
	warnings = append(warnings, msgResult.Warnings...)

	huggingfaceOpts, err := providerutils.ParseProviderOptions(
		"huggingface",
		toInterfaceMap(options.ProviderOptions),
		responsesProviderOptionsSchema,
	)
	if err != nil {
		return getArgsResult{}, err
	}

	toolResult := prepareResponsesTools(options.Tools, options.ToolChoice)
	warnings = append(warnings, toolResult.ToolWarnings...)

	baseArgs := map[string]any{
		"model": m.modelID,
		"input": msgResult.Input,
	}

	if options.Temperature != nil {
		baseArgs["temperature"] = *options.Temperature
	}
	if options.TopP != nil {
		baseArgs["top_p"] = *options.TopP
	}
	if options.MaxOutputTokens != nil {
		baseArgs["max_output_tokens"] = *options.MaxOutputTokens
	}

	// HuggingFace Responses API uses text.format for structured output.
	if jsonFmt, ok := options.ResponseFormat.(languagemodel.ResponseFormatJSON); ok && jsonFmt.Schema != nil {
		strictVal := false
		if huggingfaceOpts != nil && huggingfaceOpts.StrictJSONSchema != nil {
			strictVal = *huggingfaceOpts.StrictJSONSchema
		}

		formatObj := map[string]any{
			"type":   "json_schema",
			"strict": strictVal,
			"name":   "response",
			"schema": jsonFmt.Schema,
		}
		if jsonFmt.Name != nil {
			formatObj["name"] = *jsonFmt.Name
		}
		if jsonFmt.Description != nil {
			formatObj["description"] = *jsonFmt.Description
		}

		baseArgs["text"] = map[string]any{
			"format": formatObj,
		}
	}

	if huggingfaceOpts != nil {
		if huggingfaceOpts.Metadata != nil {
			baseArgs["metadata"] = huggingfaceOpts.Metadata
		}
		if huggingfaceOpts.Instructions != nil {
			baseArgs["instructions"] = *huggingfaceOpts.Instructions
		}
		if huggingfaceOpts.ReasoningEffort != nil {
			baseArgs["reasoning"] = map[string]any{
				"effort": *huggingfaceOpts.ReasoningEffort,
			}
		}
	}

	if len(toolResult.Tools) > 0 {
		baseArgs["tools"] = toolResult.Tools
	}
	if toolResult.ToolChoice != nil {
		baseArgs["tool_choice"] = toolResult.ToolChoice.MarshalToolChoice()
	}

	return getArgsResult{args: baseArgs, warnings: warnings}, nil
}

// --- Response schemas (Go structs for JSON deserialization) ---

// responsesResponse is the full response from the HuggingFace responses API.
type responsesResponse struct {
	ID                string                `json:"id"`
	Model             string                `json:"model"`
	Object            string                `json:"object"`
	CreatedAt         float64               `json:"created_at"`
	Status            string                `json:"status"`
	Error             *responsesErrorObj    `json:"error"`
	Instructions      any                   `json:"instructions"`
	MaxOutputTokens   any                   `json:"max_output_tokens"`
	Metadata          any                   `json:"metadata"`
	ToolChoice        any                   `json:"tool_choice"`
	Tools             []any                 `json:"tools"`
	Temperature       float64               `json:"temperature"`
	TopP              float64               `json:"top_p"`
	IncompleteDetails *incompleteDetails    `json:"incomplete_details,omitempty"`
	Usage             *ResponsesUsage       `json:"usage,omitempty"`
	Output            []responsesOutputItem `json:"output"`
	OutputText        *string               `json:"output_text,omitempty"`
}

type responsesErrorObj struct {
	Message string `json:"message"`
}

type incompleteDetails struct {
	Reason string `json:"reason"`
}

// responsesOutputItem represents an item in the response output array.
type responsesOutputItem struct {
	Type        string                  `json:"type"`
	ID          string                  `json:"id"`
	Role        *string                 `json:"role,omitempty"`
	Status      *string                 `json:"status,omitempty"`
	Content     []responsesContentPart  `json:"content,omitempty"`
	Summary     []any                   `json:"summary,omitempty"`
	CallID      string                  `json:"call_id,omitempty"`
	Name        string                  `json:"name,omitempty"`
	Arguments   string                  `json:"arguments,omitempty"`
	Output      string                  `json:"output,omitempty"`
	ServerLabel string                  `json:"server_label,omitempty"`
	Tools       []any                   `json:"tools,omitempty"`
}

type responsesContentPart struct {
	Type        string                   `json:"type"`
	Text        string                   `json:"text"`
	Annotations []responsesAnnotation    `json:"annotations,omitempty"`
}

type responsesAnnotation struct {
	URL   string  `json:"url"`
	Title *string `json:"title,omitempty"`
}

// responsesResponseSchema is the schema for parsing the full response.
var responsesResponseSchema = &providerutils.Schema[responsesResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[responsesResponse], error) {
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[responsesResponse]{Success: false}, err
		}
		var resp responsesResponse
		if err := json.Unmarshal(jsonBytes, &resp); err != nil {
			return &providerutils.ValidationResult[responsesResponse]{Success: false}, err
		}
		return &providerutils.ValidationResult[responsesResponse]{
			Success: true,
			Value:   resp,
		}, nil
	},
}

// --- Stream chunk types ---

type streamChunk struct {
	Type           string          `json:"type"`
	OutputIndex    *int            `json:"output_index,omitempty"`
	ItemID         string          `json:"item_id,omitempty"`
	ContentIndex   *int            `json:"content_index,omitempty"`
	Delta          string          `json:"delta,omitempty"`
	Text           string          `json:"text,omitempty"`
	SequenceNumber *int            `json:"sequence_number,omitempty"`
	Item           json.RawMessage `json:"item,omitempty"`
	Response       json.RawMessage `json:"response,omitempty"`
}

type streamChunkItem struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Role        string `json:"role,omitempty"`
	Status      string `json:"status,omitempty"`
	CallID      string `json:"call_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Arguments   string `json:"arguments,omitempty"`
	Output      string `json:"output,omitempty"`
	ServerLabel string `json:"server_label,omitempty"`
}

type streamChunkCreatedResponse struct {
	ID        string  `json:"id"`
	Object    string  `json:"object"`
	CreatedAt float64 `json:"created_at"`
	Status    string  `json:"status"`
	Model     string  `json:"model"`
}

var streamChunkSchema = &providerutils.Schema[streamChunk]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[streamChunk], error) {
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[streamChunk]{Success: false}, err
		}
		var chunk streamChunk
		if err := json.Unmarshal(jsonBytes, &chunk); err != nil {
			return &providerutils.ValidationResult[streamChunk]{Success: false}, err
		}
		return &providerutils.ValidationResult[streamChunk]{
			Success: true,
			Value:   chunk,
		}, nil
	},
}

// --- DoGenerate ---

// DoGenerate implements the non-streaming generation.
func (m *ResponsesLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	argsResult, err := m.getArgs(options)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	body := make(map[string]any)
	for k, v := range argsResult.args {
		body[k] = v
	}
	body["stream"] = false

	apiURL := m.config.URL(URLOptions{
		Path:    "/responses",
		ModelID: m.modelID,
	})

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[responsesResponse]{
		URL:                       apiURL,
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(responsesResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	response := result.Value

	if response.Error != nil {
		statusCode := 400
		responseBody := ""
		if s, ok := result.RawValue.(string); ok {
			responseBody = s
		}
		return languagemodel.GenerateResult{}, errors.NewAPICallError(errors.APICallErrorOptions{
			Message:           response.Error.Message,
			URL:               apiURL,
			RequestBodyValues: body,
			StatusCode:        &statusCode,
			ResponseHeaders:   result.ResponseHeaders,
			ResponseBody:      &responseBody,
			IsRetryable:       boolPtr(false),
		})
	}

	content := []languagemodel.Content{}

	// Process output array.
	for _, part := range response.Output {
		switch part.Type {
		case "message":
			for _, contentPart := range part.Content {
				content = append(content, languagemodel.Text{
					Text: contentPart.Text,
					ProviderMetadata: shared.ProviderMetadata{
						"huggingface": jsonvalue.JSONObject{
							"itemId": part.ID,
						},
					},
				})

				if len(contentPart.Annotations) > 0 {
					for _, annotation := range contentPart.Annotations {
						genID := providerutils.GenerateId
						if m.config.GenerateID != nil {
							genID = m.config.GenerateID
						}
						content = append(content, languagemodel.SourceURL{
							ID:    genID(),
							URL:   annotation.URL,
							Title: annotation.Title,
						})
					}
				}
			}

		case "reasoning":
			for _, contentPart := range part.Content {
				content = append(content, languagemodel.Reasoning{
					Text: contentPart.Text,
					ProviderMetadata: shared.ProviderMetadata{
						"huggingface": jsonvalue.JSONObject{
							"itemId": part.ID,
						},
					},
				})
			}

		case "mcp_call":
			content = append(content, languagemodel.ToolCall{
				ToolCallID:       part.ID,
				ToolName:         part.Name,
				Input:            part.Arguments,
				ProviderExecuted: boolPtr(true),
			})

			if part.Output != "" {
				content = append(content, languagemodel.ToolResult{
					ToolCallID: part.ID,
					ToolName:   part.Name,
					Result:     part.Output,
				})
			}

		case "mcp_list_tools":
			serverLabelJSON, _ := json.Marshal(map[string]string{
				"server_label": part.ServerLabel,
			})
			content = append(content, languagemodel.ToolCall{
				ToolCallID:       part.ID,
				ToolName:         "list_tools",
				Input:            string(serverLabelJSON),
				ProviderExecuted: boolPtr(true),
			})

			if part.Tools != nil {
				content = append(content, languagemodel.ToolResult{
					ToolCallID: part.ID,
					ToolName:   "list_tools",
					Result:     map[string]any{"tools": part.Tools},
				})
			}

		case "function_call":
			content = append(content, languagemodel.ToolCall{
				ToolCallID: part.CallID,
				ToolName:   part.Name,
				Input:      part.Arguments,
			})

			if part.Output != "" {
				content = append(content, languagemodel.ToolResult{
					ToolCallID: part.CallID,
					ToolName:   part.Name,
					Result:     part.Output,
				})
			}
		}
	}

	finishReasonRaw := ""
	if response.IncompleteDetails != nil {
		finishReasonRaw = response.IncompleteDetails.Reason
	}
	if finishReasonRaw == "" {
		finishReasonRaw = "stop"
	}

	var rawFinishReason *string
	if response.IncompleteDetails != nil {
		rawFinishReason = &response.IncompleteDetails.Reason
	}

	timestamp := time.Unix(int64(response.CreatedAt), 0)

	return languagemodel.GenerateResult{
		Content: content,
		FinishReason: languagemodel.FinishReason{
			Unified: mapHuggingFaceResponsesFinishReason(finishReasonRaw),
			Raw:     rawFinishReason,
		},
		Usage: convertHuggingFaceResponsesUsage(response.Usage),
		Request: &languagemodel.GenerateResultRequest{
			Body: body,
		},
		Response: &languagemodel.GenerateResultResponse{
			ResponseMetadata: languagemodel.ResponseMetadata{
				ID:        &response.ID,
				Timestamp: &timestamp,
				ModelID:   &response.Model,
			},
			Headers: result.ResponseHeaders,
			Body:    result.RawValue,
		},
		ProviderMetadata: shared.ProviderMetadata{
			"huggingface": jsonvalue.JSONObject{
				"responseId": response.ID,
			},
		},
		Warnings: argsResult.warnings,
	}, nil
}

// --- DoStream ---

// DoStream implements streaming generation.
func (m *ResponsesLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	argsResult, err := m.getArgs(options)
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	body := make(map[string]any)
	for k, v := range argsResult.args {
		body[k] = v
	}
	body["stream"] = true

	apiURL := m.config.URL(URLOptions{
		Path:    "/responses",
		ModelID: m.modelID,
	})

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[<-chan providerutils.ParseResult[streamChunk]]{
		URL:                       apiURL,
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(streamChunkSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	outCh := make(chan languagemodel.StreamPart)

	go func() {
		defer close(outCh)

		finishReason := languagemodel.FinishReason{
			Unified: languagemodel.FinishReasonOther,
			Raw:     nil,
		}
		var responseID *string
		var usage *ResponsesUsage

		// Send stream start.
		outCh <- languagemodel.StreamPartStreamStart{
			Warnings: argsResult.warnings,
		}

		for chunk := range result.Value {
			if !chunk.Success {
				finishReason = languagemodel.FinishReason{
					Unified: languagemodel.FinishReasonError,
					Raw:     nil,
				}
				outCh <- languagemodel.StreamPartError{Error: chunk.Error}
				continue
			}

			value := chunk.Value

			switch value.Type {
			case "response.created":
				var createdResp streamChunkCreatedResponse
				if err := json.Unmarshal(value.Response, &createdResp); err == nil {
					responseID = &createdResp.ID
					ts := time.Unix(int64(createdResp.CreatedAt), 0)
					outCh <- languagemodel.StreamPartResponseMetadata{
						ResponseMetadata: languagemodel.ResponseMetadata{
							ID:        &createdResp.ID,
							Timestamp: &ts,
							ModelID:   &createdResp.Model,
						},
					}
				}

			case "response.output_item.added":
				var item streamChunkItem
				if err := json.Unmarshal(value.Item, &item); err == nil {
					switch item.Type {
					case "message":
						if item.Role == "assistant" {
							outCh <- languagemodel.StreamPartTextStart{
								ID: item.ID,
								ProviderMetadata: shared.ProviderMetadata{
									"huggingface": jsonvalue.JSONObject{
										"itemId": item.ID,
									},
								},
							}
						}
					case "function_call":
						outCh <- languagemodel.StreamPartToolInputStart{
							ID:       item.CallID,
							ToolName: item.Name,
						}
					case "reasoning":
						outCh <- languagemodel.StreamPartReasoningStart{
							ID: item.ID,
							ProviderMetadata: shared.ProviderMetadata{
								"huggingface": jsonvalue.JSONObject{
									"itemId": item.ID,
								},
							},
						}
					}
				}

			case "response.output_item.done":
				var item streamChunkItem
				if err := json.Unmarshal(value.Item, &item); err == nil {
					switch item.Type {
					case "message":
						if item.Role == "assistant" {
							outCh <- languagemodel.StreamPartTextEnd{
								ID: item.ID,
							}
						}
					case "function_call":
						outCh <- languagemodel.StreamPartToolInputEnd{
							ID: item.CallID,
						}
						outCh <- languagemodel.ToolCall{
							ToolCallID: item.CallID,
							ToolName:   item.Name,
							Input:      item.Arguments,
						}
						if item.Output != "" {
							outCh <- languagemodel.ToolResult{
								ToolCallID: item.CallID,
								ToolName:   item.Name,
								Result:     item.Output,
							}
						}
					}
				}

			case "response.completed":
				var completedResp responsesResponse
				if err := json.Unmarshal(value.Response, &completedResp); err == nil {
					responseID = &completedResp.ID

					finishReasonRaw := "stop"
					if completedResp.IncompleteDetails != nil {
						finishReasonRaw = completedResp.IncompleteDetails.Reason
					}

					var rawStr *string
					if completedResp.IncompleteDetails != nil {
						rawStr = &completedResp.IncompleteDetails.Reason
					}

					finishReason = languagemodel.FinishReason{
						Unified: mapHuggingFaceResponsesFinishReason(finishReasonRaw),
						Raw:     rawStr,
					}
					if completedResp.Usage != nil {
						usage = completedResp.Usage
					}
				}

			case "response.reasoning_text.delta":
				outCh <- languagemodel.StreamPartReasoningDelta{
					ID:    value.ItemID,
					Delta: value.Delta,
				}

			case "response.reasoning_text.done":
				outCh <- languagemodel.StreamPartReasoningEnd{
					ID: value.ItemID,
				}

			case "response.output_text.delta":
				outCh <- languagemodel.StreamPartTextDelta{
					ID:    value.ItemID,
					Delta: value.Delta,
				}
			}
		}

		// Flush: send finish.
		outCh <- languagemodel.StreamPartFinish{
			FinishReason: finishReason,
			Usage:        convertHuggingFaceResponsesUsage(usage),
			ProviderMetadata: shared.ProviderMetadata{
				"huggingface": jsonvalue.JSONObject{
					"responseId": responseID,
				},
			},
		}
	}()

	return languagemodel.StreamResult{
		Stream: outCh,
		Request: &languagemodel.StreamResultRequest{
			Body: body,
		},
		Response: &languagemodel.StreamResultResponse{
			Headers: result.ResponseHeaders,
		},
	}, nil
}

// --- Helpers ---

func boolPtr(b bool) *bool {
	return &b
}

// Compile-time check that ResponsesLanguageModel implements the interface.
var _ languagemodel.LanguageModel = (*ResponsesLanguageModel)(nil)

// stringPtr returns a pointer to the given string.
func stringPtr(s string) *string {
	return &s
}

// Helper to dereference a *string with a default if nil.
func derefString(s *string, def string) string {
	if s != nil {
		return *s
	}
	return def
}

// Unused but kept for parity: intPtr returns a pointer to the given int.
func intPtr(i int) *int {
	return &i
}

// toInterfaceMap converts shared.ProviderOptions (map[string]map[string]any) to
// map[string]interface{} for use with providerutils.ParseProviderOptions.
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

// convertOptionalHeaders converts map[string]*string to map[string]string,
// skipping nil values.
func convertOptionalHeaders(headers map[string]*string) map[string]string {
	if headers == nil {
		return nil
	}
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}

// failedResponseHandler wraps the typed FailedResponseHandler into a
// ResponseHandler[error] as required by PostJsonToApi.
var failedResponseHandler providerutils.ResponseHandler[error] = func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
	res, err := FailedResponseHandler(opts)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return &providerutils.ResponseHandlerResult[error]{
		Value:           res.Value,
		RawValue:        res.RawValue,
		ResponseHeaders: res.ResponseHeaders,
	}, nil
}

// Suppress unused warnings for helper functions.
var (
	_ = stringPtr
	_ = derefString
	_ = intPtr
	_ = fmt.Sprintf
)
