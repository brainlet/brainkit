package messages

// ── Requests ──

type AiGenerateMsg struct {
	Model    string          `json:"model"`
	Prompt   string          `json:"prompt,omitempty"`
	Messages []AiChatMessage `json:"messages,omitempty"`
	Tools    []string        `json:"tools,omitempty"`
	Schema   any             `json:"schema,omitempty"`
}

func (AiGenerateMsg) BusTopic() string { return "ai.generate" }

type AiStreamMsg struct {
	Model    string          `json:"model"`
	Prompt   string          `json:"prompt,omitempty"`
	Messages []AiChatMessage `json:"messages,omitempty"`
	StreamTo string          `json:"streamTo"`
}

func (AiStreamMsg) BusTopic() string { return "ai.stream" }

type AiEmbedMsg struct {
	Model string `json:"model"`
	Value string `json:"value"`
}

func (AiEmbedMsg) BusTopic() string { return "ai.embed" }

type AiEmbedManyMsg struct {
	Model  string   `json:"model"`
	Values []string `json:"values"`
}

func (AiEmbedManyMsg) BusTopic() string { return "ai.embedMany" }

type AiGenerateObjectMsg struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Schema any    `json:"schema"`
}

func (AiGenerateObjectMsg) BusTopic() string { return "ai.generateObject" }

// ── Responses ──

type AiGenerateResp struct {
	ResultMeta
	Text      string       `json:"text"`
	ToolCalls []AiToolCall `json:"toolCalls,omitempty"`
	Usage     AiUsage      `json:"usage"`
}


type AiEmbedResp struct {
	ResultMeta
	Embedding []float64 `json:"embedding"`
}


type AiEmbedManyResp struct {
	ResultMeta
	Embeddings [][]float64 `json:"embeddings"`
}


type AiGenerateObjectResp struct {
	ResultMeta
	Object any `json:"object"`
}


// ── Shared types ──

type AiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AiToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args any    `json:"args"`
}

type AiUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}
