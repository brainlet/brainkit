// Ported from: packages/togetherai/src/togetherai-embedding-options.ts
package togetherai

// https://docs.together.ai/docs/serverless-models#embedding-models

// TogetherAIEmbeddingModelID is the type for Together AI embedding model identifiers.
// Known model IDs are provided as constants; any string is accepted.
type TogetherAIEmbeddingModelID = string

// Known Together AI embedding model IDs.
const (
	EmbeddingModelM2Bert80M2kRetrieval   TogetherAIEmbeddingModelID = "togethercomputer/m2-bert-80M-2k-retrieval"
	EmbeddingModelM2Bert80M32kRetrieval  TogetherAIEmbeddingModelID = "togethercomputer/m2-bert-80M-32k-retrieval"
	EmbeddingModelM2Bert80M8kRetrieval   TogetherAIEmbeddingModelID = "togethercomputer/m2-bert-80M-8k-retrieval"
	EmbeddingModelUAELargeV1             TogetherAIEmbeddingModelID = "WhereIsAI/UAE-Large-V1"
	EmbeddingModelBGELargeEnV15          TogetherAIEmbeddingModelID = "BAAI/bge-large-en-v1.5"
	EmbeddingModelBGEBaseEnV15           TogetherAIEmbeddingModelID = "BAAI/bge-base-en-v1.5"
	EmbeddingModelMSMarcoBertBaseDotV5   TogetherAIEmbeddingModelID = "sentence-transformers/msmarco-bert-base-dot-v5"
	EmbeddingModelBertBaseUncased        TogetherAIEmbeddingModelID = "bert-base-uncased"
)
