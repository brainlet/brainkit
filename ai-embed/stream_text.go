package aiembed

// StreamTextParams configures a streamText call.
type StreamTextParams struct {
	Model    Model `json:"model"`
	CallSettings
	Prompt          string                             `json:"prompt,omitempty"`
	System          string                             `json:"system,omitempty"`
	Messages        []Message                          `json:"messages,omitempty"`
	Tools           map[string]Tool                    `json:"tools,omitempty"`
	ToolChoice      *ToolChoice                        `json:"toolChoice,omitempty"`
	MaxSteps        int                                `json:"maxSteps,omitempty"`
	ProviderOptions map[string]map[string]interface{}  `json:"providerOptions,omitempty"`
	OnToken         func(token string)                 `json:"-"`
	OnToolCall      func(ToolCall)                     `json:"-"`
	OnStepFinish    func(StepResult)                   `json:"-"`
	OnFinish        func(GenerateTextResult)           `json:"-"`
	OnError         func(error)                        `json:"-"`
}

// StreamTextResult is returned by StreamText.
type StreamTextResult struct {
	Text         string       `json:"text"`
	FinishReason FinishReason `json:"finishReason"`
	Usage        Usage        `json:"usage"`
	ToolCalls    []ToolCall   `json:"toolCalls,omitempty"`
	ToolResults  []ToolResult `json:"toolResults,omitempty"`
	Steps        []StepResult `json:"steps,omitempty"`
	Response     ResponseMeta `json:"response"`
}
