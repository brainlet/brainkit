// Ported from: packages/google/src/google-prepare-tools.ts
package google

import (
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// FunctionCallingConfig represents the function calling configuration.
type FunctionCallingConfig struct {
	Mode                 string   `json:"mode"`
	AllowedFunctionNames []string `json:"allowedFunctionNames,omitempty"`
}

// ToolConfig represents the tool configuration.
type ToolConfig struct {
	FunctionCallingConfig *FunctionCallingConfig `json:"functionCallingConfig,omitempty"`
	RetrievalConfig       *RetrievalConfig       `json:"retrievalConfig,omitempty"`
}

// FunctionDeclaration represents a function declaration for the Google API.
type FunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// PrepareToolsResult is the result of PrepareTools.
type PrepareToolsResult struct {
	Tools        []map[string]any `json:"tools,omitempty"`
	ToolConfig   *ToolConfig      `json:"toolConfig,omitempty"`
	ToolWarnings []shared.Warning
}

// PrepareTools converts language model tools to the Google Generative AI format.
func PrepareTools(
	tools []languagemodel.Tool,
	toolChoice languagemodel.ToolChoice,
	modelID string,
) (*PrepareToolsResult, error) {
	// When the tools slice is empty, treat as nil to prevent errors.
	if len(tools) == 0 {
		tools = nil
	}

	var toolWarnings []shared.Warning

	isLatest := modelID == "gemini-flash-latest" ||
		modelID == "gemini-flash-lite-latest" ||
		modelID == "gemini-pro-latest"

	isGemini2orNewer := strings.Contains(modelID, "gemini-2") ||
		strings.Contains(modelID, "gemini-3") ||
		strings.Contains(modelID, "nano-banana") ||
		isLatest

	supportsFileSearch := strings.Contains(modelID, "gemini-2.5") ||
		strings.Contains(modelID, "gemini-3")

	if tools == nil {
		return &PrepareToolsResult{
			ToolWarnings: toolWarnings,
		}, nil
	}

	// Check for mixed tool types and add warnings.
	hasFunctionTools := false
	hasProviderTools := false
	for _, tool := range tools {
		switch tool.(type) {
		case languagemodel.FunctionTool:
			hasFunctionTools = true
		case languagemodel.ProviderTool:
			hasProviderTools = true
		}
	}

	if hasFunctionTools && hasProviderTools {
		toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
			Feature: "combination of function and provider-defined tools",
		})
	}

	if hasProviderTools {
		var googleTools []map[string]any

		for _, tool := range tools {
			pt, ok := tool.(languagemodel.ProviderTool)
			if !ok {
				continue
			}
			switch pt.ID {
			case "google.google_search":
				if isGemini2orNewer {
					entry := map[string]any{"googleSearch": pt.Args}
					if entry["googleSearch"] == nil {
						entry["googleSearch"] = map[string]any{}
					}
					googleTools = append(googleTools, entry)
				} else {
					details := "Google Search requires Gemini 2.0 or newer."
					toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
						Feature: "provider-defined tool " + pt.ID,
						Details: &details,
					})
				}
			case "google.enterprise_web_search":
				if isGemini2orNewer {
					googleTools = append(googleTools, map[string]any{"enterpriseWebSearch": map[string]any{}})
				} else {
					details := "Enterprise Web Search requires Gemini 2.0 or newer."
					toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
						Feature: "provider-defined tool " + pt.ID,
						Details: &details,
					})
				}
			case "google.url_context":
				if isGemini2orNewer {
					googleTools = append(googleTools, map[string]any{"urlContext": map[string]any{}})
				} else {
					details := "The URL context tool is not supported with other Gemini models than Gemini 2."
					toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
						Feature: "provider-defined tool " + pt.ID,
						Details: &details,
					})
				}
			case "google.code_execution":
				if isGemini2orNewer {
					googleTools = append(googleTools, map[string]any{"codeExecution": map[string]any{}})
				} else {
					details := "The code execution tools is not supported with other Gemini models than Gemini 2."
					toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
						Feature: "provider-defined tool " + pt.ID,
						Details: &details,
					})
				}
			case "google.file_search":
				if supportsFileSearch {
					entry := map[string]any{"fileSearch": pt.Args}
					if entry["fileSearch"] == nil {
						entry["fileSearch"] = map[string]any{}
					}
					googleTools = append(googleTools, entry)
				} else {
					details := "The file search tool is only supported with Gemini 2.5 models and Gemini 3 models."
					toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
						Feature: "provider-defined tool " + pt.ID,
						Details: &details,
					})
				}
			case "google.vertex_rag_store":
				if isGemini2orNewer {
					var ragCorpus string
					var topK any
					if pt.Args != nil {
						if rc, ok := pt.Args["ragCorpus"].(string); ok {
							ragCorpus = rc
						}
						topK = pt.Args["topK"]
					}
					googleTools = append(googleTools, map[string]any{
						"retrieval": map[string]any{
							"vertex_rag_store": map[string]any{
								"rag_resources": map[string]any{
									"rag_corpus": ragCorpus,
								},
								"similarity_top_k": topK,
							},
						},
					})
				} else {
					details := "The RAG store tool is not supported with other Gemini models than Gemini 2."
					toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
						Feature: "provider-defined tool " + pt.ID,
						Details: &details,
					})
				}
			case "google.google_maps":
				if isGemini2orNewer {
					googleTools = append(googleTools, map[string]any{"googleMaps": map[string]any{}})
				} else {
					details := "The Google Maps grounding tool is not supported with Gemini models other than Gemini 2 or newer."
					toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
						Feature: "provider-defined tool " + pt.ID,
						Details: &details,
					})
				}
			default:
				toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
					Feature: "provider-defined tool " + pt.ID,
				})
			}
		}

		var resultTools []map[string]any
		if len(googleTools) > 0 {
			resultTools = googleTools
		}
		return &PrepareToolsResult{
			Tools:        resultTools,
			ToolWarnings: toolWarnings,
		}, nil
	}

	// Function tools
	var functionDeclarations []map[string]any
	hasStrictTools := false
	for _, tool := range tools {
		switch t := tool.(type) {
		case languagemodel.FunctionTool:
			desc := ""
			if t.Description != nil {
				desc = *t.Description
			}
			decl := map[string]any{
				"name":        t.Name,
				"description": desc,
			}
			params := ConvertJSONSchemaToOpenAPISchema(t.InputSchema, true)
			if params != nil {
				decl["parameters"] = params
			}
			functionDeclarations = append(functionDeclarations, decl)
			if t.Strict != nil && *t.Strict {
				hasStrictTools = true
			}
		default:
			toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
				Feature: "unsupported tool type",
			})
		}
	}

	if toolChoice == nil {
		var tc *ToolConfig
		if hasStrictTools {
			tc = &ToolConfig{
				FunctionCallingConfig: &FunctionCallingConfig{Mode: "VALIDATED"},
			}
		}
		return &PrepareToolsResult{
			Tools:        []map[string]any{{"functionDeclarations": functionDeclarations}},
			ToolConfig:   tc,
			ToolWarnings: toolWarnings,
		}, nil
	}

	switch tc := toolChoice.(type) {
	case languagemodel.ToolChoiceAuto:
		mode := "AUTO"
		if hasStrictTools {
			mode = "VALIDATED"
		}
		return &PrepareToolsResult{
			Tools: []map[string]any{{"functionDeclarations": functionDeclarations}},
			ToolConfig: &ToolConfig{
				FunctionCallingConfig: &FunctionCallingConfig{Mode: mode},
			},
			ToolWarnings: toolWarnings,
		}, nil
	case languagemodel.ToolChoiceNone:
		return &PrepareToolsResult{
			Tools: []map[string]any{{"functionDeclarations": functionDeclarations}},
			ToolConfig: &ToolConfig{
				FunctionCallingConfig: &FunctionCallingConfig{Mode: "NONE"},
			},
			ToolWarnings: toolWarnings,
		}, nil
	case languagemodel.ToolChoiceRequired:
		mode := "ANY"
		if hasStrictTools {
			mode = "VALIDATED"
		}
		return &PrepareToolsResult{
			Tools: []map[string]any{{"functionDeclarations": functionDeclarations}},
			ToolConfig: &ToolConfig{
				FunctionCallingConfig: &FunctionCallingConfig{Mode: mode},
			},
			ToolWarnings: toolWarnings,
		}, nil
	case languagemodel.ToolChoiceTool:
		mode := "ANY"
		if hasStrictTools {
			mode = "VALIDATED"
		}
		return &PrepareToolsResult{
			Tools: []map[string]any{{"functionDeclarations": functionDeclarations}},
			ToolConfig: &ToolConfig{
				FunctionCallingConfig: &FunctionCallingConfig{
					Mode:                 mode,
					AllowedFunctionNames: []string{tc.ToolName},
				},
			},
			ToolWarnings: toolWarnings,
		}, nil
	default:
		return nil, errors.NewUnsupportedFunctionalityError("unsupported tool choice type", "")
	}
}
