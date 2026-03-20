package aiembed

// EmbedParams configures an embed call.
type EmbedParams struct {
	Model Model  `json:"model"`
	Value string `json:"value"`
}

// EmbedResult is returned by Embed.
type EmbedResult struct {
	Embedding []float64  `json:"embedding"`
	Usage     EmbedUsage `json:"usage"`
}

// EmbedManyParams configures an embedMany call.
type EmbedManyParams struct {
	Model  Model    `json:"model"`
	Values []string `json:"values"`
}

// EmbedManyResult is returned by EmbedMany.
type EmbedManyResult struct {
	Embeddings [][]float64 `json:"embeddings"`
	Usage      EmbedUsage  `json:"usage"`
}
