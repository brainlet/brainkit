// Ported from: packages/xai/src/responses/xai-responses-language-model.ts
package xai

import (
	"fmt"
	"regexp"

	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// XaiResponsesConfig configures the xAI responses language model.
type XaiResponsesConfig struct {
	Provider   string
	BaseURL    string
	Headers    func() map[string]string
	GenerateID providerutils.IdGenerator
	Fetch      providerutils.FetchFunction
}

// XaiResponsesLanguageModel implements the LanguageModel interface for xAI responses API.
type XaiResponsesLanguageModel struct {
	specificationVersion string
	modelId              XaiResponsesModelId
	config               XaiResponsesConfig
	supportedUrls        map[string][]*regexp.Regexp
}

// NewXaiResponsesLanguageModel creates a new xAI responses language model.
func NewXaiResponsesLanguageModel(modelId XaiResponsesModelId, config XaiResponsesConfig) *XaiResponsesLanguageModel {
	return &XaiResponsesLanguageModel{
		specificationVersion: "v3",
		modelId:              modelId,
		config:               config,
		supportedUrls: map[string][]*regexp.Regexp{
			"image/*": {regexp.MustCompile(`^https?://.*$`)},
		},
	}
}

// SpecificationVersion returns the language model interface version.
func (m *XaiResponsesLanguageModel) SpecificationVersion() string {
	return m.specificationVersion
}

// Provider returns the provider name.
func (m *XaiResponsesLanguageModel) Provider() string {
	return m.config.Provider
}

// ModelID returns the model ID.
func (m *XaiResponsesLanguageModel) ModelID() string {
	return m.modelId
}

// SupportedUrls returns the supported URL patterns by media type.
func (m *XaiResponsesLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return m.supportedUrls, nil
}

// getArgs builds request arguments for the responses API.
type responsesGetArgsResult struct {
	Args                  map[string]interface{}
	Warnings              []shared.Warning
	WebSearchToolName     *string
	XSearchToolName       *string
	CodeExecutionToolName *string
	McpToolName           *string
	FileSearchToolName    *string
}

func (m *XaiResponsesLanguageModel) getArgs(options languagemodel.CallOptions) (responsesGetArgsResult, error) {
	var warnings []shared.Warning

	opts, err := providerutils.ParseProviderOptions("xai", providerOptionsToMap(options.ProviderOptions), xaiLanguageModelResponsesOptionsSchema)
	if err != nil {
		return responsesGetArgsResult{}, err
	}
	if opts == nil {
		opts = &XaiLanguageModelResponsesOptions{}
	}

	if options.StopSequences != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "stopSequences"})
	}

	// Find provider tool names
	var webSearchToolName *string
	var xSearchToolName *string
	var codeExecutionToolName *string
	var mcpToolName *string
	var fileSearchToolName *string

	for _, tool := range options.Tools {
		if pt, ok := tool.(languagemodel.ProviderTool); ok {
			switch pt.ID {
			case "xai.web_search":
				webSearchToolName = &pt.Name
			case "xai.x_search":
				xSearchToolName = &pt.Name
			case "xai.code_execution":
				codeExecutionToolName = &pt.Name
			case "xai.mcp":
				mcpToolName = &pt.Name
			case "xai.file_search":
				fileSearchToolName = &pt.Name
			}
		}
	}

	inputResult, err := convertToXaiResponsesInput(options.Prompt)
	if err != nil {
		return responsesGetArgsResult{}, err
	}
	warnings = append(warnings, inputResult.InputWarnings...)

	toolResult := prepareResponsesTools(options.Tools, options.ToolChoice)
	warnings = append(warnings, toolResult.Warnings...)

	// Build include array
	var include XaiResponsesIncludeOptions
	if opts.Include != nil {
		include = make([]string, len(opts.Include))
		copy(include, opts.Include)
	}

	if opts.Store != nil && !*opts.Store {
		if include == nil {
			include = []string{"reasoning.encrypted_content"}
		} else {
			include = append(include, "reasoning.encrypted_content")
		}
	}

	baseArgs := map[string]interface{}{
		"model": m.modelId,
		"input": inputResult.Input,
	}

	if (opts.Logprobs != nil && *opts.Logprobs) || opts.TopLogprobs != nil {
		baseArgs["logprobs"] = true
	}
	if opts.TopLogprobs != nil {
		baseArgs["top_logprobs"] = *opts.TopLogprobs
	}
	if options.MaxOutputTokens != nil {
		baseArgs["max_output_tokens"] = *options.MaxOutputTokens
	}
	if options.Temperature != nil {
		baseArgs["temperature"] = *options.Temperature
	}
	if options.TopP != nil {
		baseArgs["top_p"] = *options.TopP
	}
	if options.Seed != nil {
		baseArgs["seed"] = *options.Seed
	}

	// Response format
	if rf, ok := options.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
		if rf.Schema != nil {
			name := "response"
			if rf.Name != nil {
				name = *rf.Name
			}
			textFormat := map[string]interface{}{
				"type":   "json_schema",
				"strict": true,
				"name":   name,
				"schema": rf.Schema,
			}
			if rf.Description != nil {
				textFormat["description"] = *rf.Description
			}
			baseArgs["text"] = map[string]interface{}{
				"format": textFormat,
			}
		} else {
			baseArgs["text"] = map[string]interface{}{
				"format": map[string]interface{}{
					"type": "json_object",
				},
			}
		}
	}

	if opts.ReasoningEffort != nil {
		baseArgs["reasoning"] = map[string]interface{}{
			"effort": *opts.ReasoningEffort,
		}
	}

	if opts.Store != nil && !*opts.Store {
		baseArgs["store"] = false
	}

	if include != nil {
		baseArgs["include"] = include
	}

	if opts.PreviousResponseId != nil {
		baseArgs["previous_response_id"] = *opts.PreviousResponseId
	}

	if toolResult.Tools != nil && len(toolResult.Tools) > 0 {
		baseArgs["tools"] = toolResult.Tools
	}

	if toolResult.ToolChoice != nil {
		baseArgs["tool_choice"] = toolResult.ToolChoice
	}

	return responsesGetArgsResult{
		Args:                  baseArgs,
		Warnings:              warnings,
		WebSearchToolName:     webSearchToolName,
		XSearchToolName:       xSearchToolName,
		CodeExecutionToolName: codeExecutionToolName,
		McpToolName:           mcpToolName,
		FileSearchToolName:    fileSearchToolName,
	}, nil
}

// DoGenerate generates a language model output (non-streaming).
func (m *XaiResponsesLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	argsResult, err := m.getArgs(options)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	body := argsResult.Args
	warnings := argsResult.Warnings

	baseURL := m.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.x.ai/v1"
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[XaiResponsesResponse]{
		URL:                       fmt.Sprintf("%s/responses", baseURL),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), headersToStringMap(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     xaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(xaiResponsesResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	response := result.Value
	var content []languagemodel.Content

	webSearchSubTools := []string{"web_search", "web_search_with_snippets", "browse_page"}
	xSearchSubTools := []string{"x_user_search", "x_keyword_search", "x_semantic_search", "x_thread_fetch"}

	for _, part := range response.Output {
		if part.Type == "file_search_call" {
			toolName := "file_search"
			if argsResult.FileSearchToolName != nil {
				toolName = *argsResult.FileSearchToolName
			}

			provExec := true
			content = append(content, languagemodel.ToolCall{
				ToolCallID:       part.ID,
				ToolName:         toolName,
				Input:            "",
				ProviderExecuted: &provExec,
			})

			resultData := map[string]interface{}{
				"queries": coalesceStringSlice(part.Queries),
			}
			if part.Results != nil {
				var mappedResults []map[string]interface{}
				for _, r := range part.Results {
					mappedResults = append(mappedResults, map[string]interface{}{
						"fileId":   r.FileID,
						"filename": r.Filename,
						"score":    r.Score,
						"text":     r.Text,
					})
				}
				resultData["results"] = mappedResults
			} else {
				resultData["results"] = nil
			}

			content = append(content, languagemodel.ToolResult{
				ToolCallID: part.ID,
				ToolName:   toolName,
				Result:     resultData,
			})
			continue
		}

		if part.Type == "web_search_call" || part.Type == "x_search_call" ||
			part.Type == "code_interpreter_call" || part.Type == "code_execution_call" ||
			part.Type == "view_image_call" || part.Type == "view_x_video_call" ||
			part.Type == "custom_tool_call" || part.Type == "mcp_call" {

			toolName := coalesceStrPtr(part.Name, "")
			partName := coalesceStrPtr(part.Name, "")

			if contains(webSearchSubTools, partName) || part.Type == "web_search_call" {
				toolName = "web_search"
				if argsResult.WebSearchToolName != nil {
					toolName = *argsResult.WebSearchToolName
				}
			} else if contains(xSearchSubTools, partName) || part.Type == "x_search_call" {
				toolName = "x_search"
				if argsResult.XSearchToolName != nil {
					toolName = *argsResult.XSearchToolName
				}
			} else if partName == "code_execution" || part.Type == "code_interpreter_call" || part.Type == "code_execution_call" {
				toolName = "code_execution"
				if argsResult.CodeExecutionToolName != nil {
					toolName = *argsResult.CodeExecutionToolName
				}
			} else if part.Type == "mcp_call" {
				toolName = "mcp"
				if argsResult.McpToolName != nil {
					toolName = *argsResult.McpToolName
				} else if partName != "" {
					toolName = partName
				}
			}

			var toolInput string
			if part.Type == "custom_tool_call" {
				toolInput = coalesceStrPtr(part.Input, "")
			} else if part.Type == "mcp_call" {
				toolInput = coalesceStrPtr(part.Arguments, "")
			} else {
				toolInput = coalesceStrPtr(part.Arguments, "")
			}

			provExec := true
			content = append(content, languagemodel.ToolCall{
				ToolCallID:       part.ID,
				ToolName:         toolName,
				Input:            toolInput,
				ProviderExecuted: &provExec,
			})
			continue
		}

		switch part.Type {
		case "message":
			for _, contentPart := range part.Content {
				if contentPart.Text != nil && *contentPart.Text != "" {
					content = append(content, languagemodel.Text{
						Text: *contentPart.Text,
					})
				}

				if len(contentPart.Annotations) > 0 {
					for _, annotation := range contentPart.Annotations {
						if annotation.Type == "url_citation" && annotation.URL != "" {
							title := annotation.URL
							if annotation.Title != nil {
								title = *annotation.Title
							}
							content = append(content, languagemodel.SourceURL{
								ID:    m.config.GenerateID(),
								URL:   annotation.URL,
								Title: &title,
							})
						}
					}
				}
			}

		case "function_call":
			content = append(content, languagemodel.ToolCall{
				ToolCallID: part.CallID,
				ToolName:   coalesceStrPtr(part.Name, ""),
				Input:      coalesceStrPtr(part.Arguments, ""),
			})

		case "reasoning":
			var summaryTexts []string
			for _, s := range part.Summary {
				if s.Text != "" {
					summaryTexts = append(summaryTexts, s.Text)
				}
			}

			if len(summaryTexts) > 0 {
				reasoningText := ""
				for _, t := range summaryTexts {
					reasoningText += t
				}

				if part.EncryptedContent != nil || part.ID != "" {
					xaiMeta := jsonvalue.JSONObject{}
					if part.EncryptedContent != nil {
						xaiMeta["reasoningEncryptedContent"] = *part.EncryptedContent
					}
					if part.ID != "" {
						xaiMeta["itemId"] = part.ID
					}
					provMeta := shared.ProviderMetadata{
						"xai": xaiMeta,
					}
					content = append(content, languagemodel.Reasoning{
						Text:             reasoningText,
						ProviderMetadata: provMeta,
					})
				} else {
					content = append(content, languagemodel.Reasoning{
						Text: reasoningText,
					})
				}
			}
		}
	}

	var usageResult languagemodel.Usage
	if response.Usage != nil {
		usageResult = convertXaiResponsesUsage(*response.Usage)
	} else {
		usageResult = zeroUsage()
	}

	metadata := getResponseMetadata(getResponseMetadataInput{
		ID:        response.ID,
		Model:     response.Model,
		CreatedAt: response.CreatedAt,
	})

	return languagemodel.GenerateResult{
		Content: content,
		FinishReason: languagemodel.FinishReason{
			Unified: mapXaiResponsesFinishReason(response.Status),
			Raw:     response.Status,
		},
		Usage: usageResult,
		Request: &languagemodel.GenerateResultRequest{
			Body: body,
		},
		Response: &languagemodel.GenerateResultResponse{
			ResponseMetadata: metadata,
			Headers:          result.ResponseHeaders,
			Body:             result.RawValue,
		},
		Warnings: warnings,
	}, nil
}

// DoStream generates a language model output (streaming).
func (m *XaiResponsesLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	argsResult, err := m.getArgs(options)
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	warnings := argsResult.Warnings

	body := make(map[string]interface{})
	for k, v := range argsResult.Args {
		body[k] = v
	}
	body["stream"] = true

	baseURL := m.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.x.ai/v1"
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[<-chan providerutils.ParseResult[XaiResponsesChunk]]{
		URL:                       fmt.Sprintf("%s/responses", baseURL),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), headersToStringMap(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     xaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(xaiResponsesChunkSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	incomingCh := result.Value
	outCh := make(chan languagemodel.StreamPart)

	go func() {
		defer close(outCh)

		finishReason := languagemodel.FinishReason{
			Unified: languagemodel.FinishReasonOther,
		}
		var usage *languagemodel.Usage
		isFirstChunk := true
		contentBlocks := make(map[string]struct{ blockType string })
		seenToolCalls := make(map[string]bool)

		// Track ongoing function calls by output_index
		ongoingToolCalls := make(map[int]*struct {
			toolName   string
			toolCallId string
		})

		activeReasoning := make(map[string]*struct {
			encryptedContent *string
		})

		webSearchSubTools := []string{"web_search", "web_search_with_snippets", "browse_page"}
		xSearchSubTools := []string{"x_user_search", "x_keyword_search", "x_semantic_search", "x_thread_fetch"}

		// Send stream-start
		outCh <- languagemodel.StreamPartStreamStart{Warnings: warnings}

		for chunk := range incomingCh {
			// Emit raw chunk if requested
			if options.IncludeRawChunks != nil && *options.IncludeRawChunks {
				outCh <- languagemodel.StreamPartRaw{RawValue: chunk.RawValue}
			}

			if !chunk.Success {
				outCh <- languagemodel.StreamPartError{Error: chunk.Error}
				continue
			}

			event := chunk.Value

			// response.created / response.in_progress
			if event.Type == "response.created" || event.Type == "response.in_progress" {
				if isFirstChunk && event.Response != nil {
					metadata := getResponseMetadata(getResponseMetadataInput{
						ID:        event.Response.ID,
						Model:     event.Response.Model,
						CreatedAt: event.Response.CreatedAt,
					})
					outCh <- languagemodel.StreamPartResponseMetadata{
						ResponseMetadata: metadata,
					}
					isFirstChunk = false
				}
				continue
			}

			// Reasoning summary part added
			if event.Type == "response.reasoning_summary_part.added" {
				blockId := fmt.Sprintf("reasoning-%s", event.ItemID)
				activeReasoning[event.ItemID] = &struct{ encryptedContent *string }{}
				outCh <- languagemodel.StreamPartReasoningStart{
					ID: blockId,
					ProviderMetadata: shared.ProviderMetadata{
						"xai": jsonvalue.JSONObject{
							"itemId": event.ItemID,
						},
					},
				}
				continue
			}

			// Reasoning summary text delta
			if event.Type == "response.reasoning_summary_text.delta" {
				blockId := fmt.Sprintf("reasoning-%s", event.ItemID)
				outCh <- languagemodel.StreamPartReasoningDelta{
					ID:    blockId,
					Delta: event.Delta,
					ProviderMetadata: shared.ProviderMetadata{
						"xai": jsonvalue.JSONObject{
							"itemId": event.ItemID,
						},
					},
				}
				continue
			}

			// Reasoning summary text done
			if event.Type == "response.reasoning_summary_text.done" {
				continue
			}

			// Reasoning text delta
			if event.Type == "response.reasoning_text.delta" {
				blockId := fmt.Sprintf("reasoning-%s", event.ItemID)

				if _, exists := activeReasoning[event.ItemID]; !exists {
					activeReasoning[event.ItemID] = &struct{ encryptedContent *string }{}
					outCh <- languagemodel.StreamPartReasoningStart{
						ID: blockId,
						ProviderMetadata: shared.ProviderMetadata{
							"xai": jsonvalue.JSONObject{
								"itemId": event.ItemID,
							},
						},
					}
				}

				outCh <- languagemodel.StreamPartReasoningDelta{
					ID:    blockId,
					Delta: event.Delta,
					ProviderMetadata: shared.ProviderMetadata{
						"xai": jsonvalue.JSONObject{
							"itemId": event.ItemID,
						},
					},
				}
				continue
			}

			// Reasoning text done
			if event.Type == "response.reasoning_text.done" {
				continue
			}

			// Output text delta
			if event.Type == "response.output_text.delta" {
				blockId := fmt.Sprintf("text-%s", event.ItemID)

				if _, exists := contentBlocks[blockId]; !exists {
					contentBlocks[blockId] = struct{ blockType string }{"text"}
					outCh <- languagemodel.StreamPartTextStart{ID: blockId}
				}

				outCh <- languagemodel.StreamPartTextDelta{
					ID:    blockId,
					Delta: event.Delta,
				}
				continue
			}

			// Output text done
			if event.Type == "response.output_text.done" {
				if len(event.Annotations) > 0 {
					for _, annotation := range event.Annotations {
						if annotation.Type == "url_citation" && annotation.URL != "" {
							title := annotation.URL
							if annotation.Title != nil {
								title = *annotation.Title
							}
							outCh <- languagemodel.SourceURL{
								ID:    m.config.GenerateID(),
								URL:   annotation.URL,
								Title: &title,
							}
						}
					}
				}
				continue
			}

			// Output text annotation added
			if event.Type == "response.output_text.annotation.added" {
				if event.Annotation != nil {
					annotation := event.Annotation
					if annotation.Type == "url_citation" && annotation.URL != "" {
						title := annotation.URL
						if annotation.Title != nil {
							title = *annotation.Title
						}
						outCh <- languagemodel.SourceURL{
							ID:    m.config.GenerateID(),
							URL:   annotation.URL,
							Title: &title,
						}
					}
				}
				continue
			}

			// Response done / completed
			if event.Type == "response.done" || event.Type == "response.completed" {
				if event.Response != nil {
					if event.Response.Usage != nil {
						u := convertXaiResponsesUsage(*event.Response.Usage)
						usage = &u
					}
					if event.Response.Status != nil {
						finishReason = languagemodel.FinishReason{
							Unified: mapXaiResponsesFinishReason(event.Response.Status),
							Raw:     event.Response.Status,
						}
					}
				}
				continue
			}

			// Custom tool call input streaming - already handled by output_item events
			if event.Type == "response.custom_tool_call_input.delta" ||
				event.Type == "response.custom_tool_call_input.done" {
				continue
			}

			// Function call arguments streaming
			if event.Type == "response.function_call_arguments.delta" {
				if tc, ok := ongoingToolCalls[event.OutputIndex]; ok && tc != nil {
					outCh <- languagemodel.StreamPartToolInputDelta{
						ID:    tc.toolCallId,
						Delta: event.Delta,
					}
				}
				continue
			}

			if event.Type == "response.function_call_arguments.done" {
				continue
			}

			// Output item added / done
			if event.Type == "response.output_item.added" || event.Type == "response.output_item.done" {
				if event.Item == nil {
					continue
				}
				part := event.Item

				if part.Type == "reasoning" {
					if event.Type == "response.output_item.done" {
						blockId := fmt.Sprintf("reasoning-%s", part.ID)

						// Emit reasoning-start if not already emitted
						if _, exists := activeReasoning[part.ID]; !exists {
							activeReasoning[part.ID] = &struct{ encryptedContent *string }{}
							provMeta := shared.ProviderMetadata{}
							if part.ID != "" {
								provMeta["xai"] = jsonvalue.JSONObject{
									"itemId": part.ID,
								}
							}
							outCh <- languagemodel.StreamPartReasoningStart{
								ID:               blockId,
								ProviderMetadata: provMeta,
							}
						}

						xaiMeta := jsonvalue.JSONObject{}
						if part.EncryptedContent != nil {
							xaiMeta["reasoningEncryptedContent"] = *part.EncryptedContent
						}
						if part.ID != "" {
							xaiMeta["itemId"] = part.ID
						}
						provMeta := shared.ProviderMetadata{
							"xai": xaiMeta,
						}

						outCh <- languagemodel.StreamPartReasoningEnd{
							ID:               blockId,
							ProviderMetadata: provMeta,
						}
						delete(activeReasoning, part.ID)
					}
					continue
				}

				if part.Type == "file_search_call" {
					toolName := "file_search"
					if argsResult.FileSearchToolName != nil {
						toolName = *argsResult.FileSearchToolName
					}

					if !seenToolCalls[part.ID] {
						seenToolCalls[part.ID] = true

						outCh <- languagemodel.StreamPartToolInputStart{
							ID:       part.ID,
							ToolName: toolName,
						}
						outCh <- languagemodel.StreamPartToolInputDelta{
							ID:    part.ID,
							Delta: "",
						}
						outCh <- languagemodel.StreamPartToolInputEnd{ID: part.ID}

						provExec := true
						outCh <- languagemodel.ToolCall{
							ToolCallID:       part.ID,
							ToolName:         toolName,
							Input:            "",
							ProviderExecuted: &provExec,
						}
					}

					if event.Type == "response.output_item.done" {
						resultData := map[string]interface{}{
							"queries": coalesceStringSlice(part.Queries),
						}
						if part.Results != nil {
							var mappedResults []map[string]interface{}
							for _, r := range part.Results {
								mappedResults = append(mappedResults, map[string]interface{}{
									"fileId":   r.FileID,
									"filename": r.Filename,
									"score":    r.Score,
									"text":     r.Text,
								})
							}
							resultData["results"] = mappedResults
						} else {
							resultData["results"] = nil
						}

						outCh <- languagemodel.ToolResult{
							ToolCallID: part.ID,
							ToolName:   toolName,
							Result:     resultData,
						}
					}
					continue
				}

				if part.Type == "web_search_call" || part.Type == "x_search_call" ||
					part.Type == "code_interpreter_call" || part.Type == "code_execution_call" ||
					part.Type == "view_image_call" || part.Type == "view_x_video_call" ||
					part.Type == "custom_tool_call" || part.Type == "mcp_call" {

					toolName := coalesceStrPtr(part.Name, "")
					partName := coalesceStrPtr(part.Name, "")

					if contains(webSearchSubTools, partName) || part.Type == "web_search_call" {
						toolName = "web_search"
						if argsResult.WebSearchToolName != nil {
							toolName = *argsResult.WebSearchToolName
						}
					} else if contains(xSearchSubTools, partName) || part.Type == "x_search_call" {
						toolName = "x_search"
						if argsResult.XSearchToolName != nil {
							toolName = *argsResult.XSearchToolName
						}
					} else if partName == "code_execution" || part.Type == "code_interpreter_call" || part.Type == "code_execution_call" {
						toolName = "code_execution"
						if argsResult.CodeExecutionToolName != nil {
							toolName = *argsResult.CodeExecutionToolName
						}
					} else if part.Type == "mcp_call" {
						toolName = "mcp"
						if argsResult.McpToolName != nil {
							toolName = *argsResult.McpToolName
						} else if partName != "" {
							toolName = partName
						}
					}

					var toolInput string
					if part.Type == "custom_tool_call" {
						toolInput = coalesceStrPtr(part.Input, "")
					} else if part.Type == "mcp_call" {
						toolInput = coalesceStrPtr(part.Arguments, "")
					} else {
						toolInput = coalesceStrPtr(part.Arguments, "")
					}

					shouldEmit := false
					if part.Type == "custom_tool_call" {
						shouldEmit = event.Type == "response.output_item.done"
					} else {
						shouldEmit = !seenToolCalls[part.ID]
					}

					if shouldEmit && !seenToolCalls[part.ID] {
						seenToolCalls[part.ID] = true

						outCh <- languagemodel.StreamPartToolInputStart{
							ID:       part.ID,
							ToolName: toolName,
						}
						outCh <- languagemodel.StreamPartToolInputDelta{
							ID:    part.ID,
							Delta: toolInput,
						}
						outCh <- languagemodel.StreamPartToolInputEnd{ID: part.ID}

						provExec := true
						outCh <- languagemodel.ToolCall{
							ToolCallID:       part.ID,
							ToolName:         toolName,
							Input:            toolInput,
							ProviderExecuted: &provExec,
						}
					}
					continue
				}

				if part.Type == "message" {
					for _, contentPart := range part.Content {
						if contentPart.Text != nil && len(*contentPart.Text) > 0 {
							blockId := fmt.Sprintf("text-%s", part.ID)

							// Only emit text if we haven't already streamed it via output_text.delta events
							if _, exists := contentBlocks[blockId]; !exists {
								contentBlocks[blockId] = struct{ blockType string }{"text"}
								outCh <- languagemodel.StreamPartTextStart{ID: blockId}
								outCh <- languagemodel.StreamPartTextDelta{
									ID:    blockId,
									Delta: *contentPart.Text,
								}
							}
						}

						if len(contentPart.Annotations) > 0 {
							for _, annotation := range contentPart.Annotations {
								if annotation.Type == "url_citation" && annotation.URL != "" {
									title := annotation.URL
									if annotation.Title != nil {
										title = *annotation.Title
									}
									outCh <- languagemodel.SourceURL{
										ID:    m.config.GenerateID(),
										URL:   annotation.URL,
										Title: &title,
									}
								}
							}
						}
					}
				} else if part.Type == "function_call" {
					if event.Type == "response.output_item.added" {
						// Track the call for function_call_arguments.delta streaming
						ongoingToolCalls[event.OutputIndex] = &struct {
							toolName   string
							toolCallId string
						}{
							toolName:   coalesceStrPtr(part.Name, ""),
							toolCallId: part.CallID,
						}

						outCh <- languagemodel.StreamPartToolInputStart{
							ID:       part.CallID,
							ToolName: coalesceStrPtr(part.Name, ""),
						}
					} else if event.Type == "response.output_item.done" {
						delete(ongoingToolCalls, event.OutputIndex)

						outCh <- languagemodel.StreamPartToolInputEnd{
							ID: part.CallID,
						}

						outCh <- languagemodel.ToolCall{
							ToolCallID: part.CallID,
							ToolName:   coalesceStrPtr(part.Name, ""),
							Input:      coalesceStrPtr(part.Arguments, ""),
						}
					}
				}
				continue
			}
		}

		// Flush: emit text-end for all text blocks
		for blockId, block := range contentBlocks {
			if block.blockType == "text" {
				outCh <- languagemodel.StreamPartTextEnd{ID: blockId}
			}
		}

		// Emit finish
		finalUsage := zeroUsage()
		if usage != nil {
			finalUsage = *usage
		}

		outCh <- languagemodel.StreamPartFinish{
			FinishReason: finishReason,
			Usage:        finalUsage,
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

// contains checks if a string slice contains a string.
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// coalesceStrPtr returns the string pointer value or a fallback.
func coalesceStrPtr(p *string, fallback string) string {
	if p != nil {
		return *p
	}
	return fallback
}

// coalesceStringSlice returns the slice or an empty slice if nil.
func coalesceStringSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
