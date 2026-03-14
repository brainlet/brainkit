package aiembed

// GenerateTextParams configures a generateText call.
type GenerateTextParams struct {
	Model    Model `json:"model"`
	CallSettings
	Prompt          string                             `json:"prompt,omitempty"`
	System          string                             `json:"system,omitempty"`
	Messages        []Message                          `json:"messages,omitempty"`
	Tools           map[string]Tool                    `json:"tools,omitempty"`
	ToolChoice      *ToolChoice                        `json:"toolChoice,omitempty"`
	MaxSteps        int                                `json:"maxSteps,omitempty"`
	ProviderOptions map[string]map[string]interface{}  `json:"providerOptions,omitempty"`
	OnStepFinish    func(StepResult)                   `json:"-"`
}

// GenerateTextResult is returned by GenerateText.
type GenerateTextResult struct {
	Text         string       `json:"text"`
	Reasoning    string       `json:"reasoning,omitempty"`
	ToolCalls    []ToolCall   `json:"toolCalls,omitempty"`
	ToolResults  []ToolResult `json:"toolResults,omitempty"`
	FinishReason FinishReason `json:"finishReason"`
	Usage        Usage        `json:"usage"`
	Steps        []StepResult `json:"steps,omitempty"`
	Response     ResponseMeta `json:"response"`
}
