package aiembed

// ProviderConfig configures an LLM provider.
type ProviderConfig struct {
	Provider string            `json:"provider,omitempty"`
	APIKey   string            `json:"apiKey,omitempty"`
	BaseURL  string            `json:"baseURL,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// Model identifies an LLM model.
type Model struct {
	ID       string          `json:"id"`
	Provider *ProviderConfig `json:"provider,omitempty"`
}

// ModelFromString creates a Model from a "provider/model" string.
func ModelFromString(id string) Model {
	return Model{ID: id}
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// FinishReason indicates why generation stopped.
type FinishReason string

const (
	FinishStop          FinishReason = "stop"
	FinishLength        FinishReason = "length"
	FinishContentFilter FinishReason = "content-filter"
	FinishToolCalls     FinishReason = "tool-calls"
	FinishError         FinishReason = "error"
	FinishOther         FinishReason = "other"
)

// ResponseMeta contains metadata about the LLM response.
type ResponseMeta struct {
	ID        string            `json:"id"`
	ModelID   string            `json:"modelId"`
	Timestamp string            `json:"timestamp,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// CallSettings configures model sampling and request behavior.
type CallSettings struct {
	MaxTokens        int      `json:"maxTokens,omitempty"`
	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"topP,omitempty"`
	TopK             *int     `json:"topK,omitempty"`
	PresencePenalty  *float64 `json:"presencePenalty,omitempty"`
	FrequencyPenalty *float64 `json:"frequencyPenalty,omitempty"`
	StopSequences    []string `json:"stopSequences,omitempty"`
	Seed             *int     `json:"seed,omitempty"`
	MaxRetries       int      `json:"maxRetries,omitempty"`
}

// StepResult contains the result of a single LLM call step.
type StepResult struct {
	Text         string       `json:"text"`
	Reasoning    string       `json:"reasoning,omitempty"`
	ToolCalls    []ToolCall   `json:"toolCalls,omitempty"`
	ToolResults  []ToolResult `json:"toolResults,omitempty"`
	FinishReason FinishReason `json:"finishReason"`
	Usage        Usage        `json:"usage"`
	StepType     string       `json:"stepType"`
	IsContinued  bool         `json:"isContinued"`
	Response     ResponseMeta `json:"response"`
}

// EmbedUsage tracks embedding token consumption.
type EmbedUsage struct {
	Tokens int `json:"tokens"`
}

// Float64 returns a pointer to v. Use for optional CallSettings fields.
func Float64(v float64) *float64 { return &v }

// Int returns a pointer to v. Use for optional CallSettings fields.
func Int(v int) *int { return &v }

// resolveModel splits a "provider/model" string and resolves provider config.
func resolveModel(m Model, defaultProvider *ProviderConfig, envVars map[string]string) (provider, modelID, apiKey, baseURL string) {
	parts := splitModelID(m.ID)
	provider = parts[0]
	modelID = parts[1]

	if m.Provider != nil {
		apiKey = m.Provider.APIKey
		baseURL = m.Provider.BaseURL
	} else if defaultProvider != nil {
		apiKey = defaultProvider.APIKey
		baseURL = defaultProvider.BaseURL
	}

	if apiKey == "" && envVars != nil {
		switch provider {
		case "openai":
			apiKey = envVars["OPENAI_API_KEY"]
		case "anthropic":
			apiKey = envVars["ANTHROPIC_API_KEY"]
		case "google":
			apiKey = envVars["GOOGLE_API_KEY"]
		}
	}

	return
}

func splitModelID(id string) [2]string {
	for i, c := range id {
		if c == '/' {
			return [2]string{id[:i], id[i+1:]}
		}
	}
	return [2]string{"openai", id}
}
