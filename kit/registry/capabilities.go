package registry

// AIProviderCapabilities describes what model types a provider supports.
type AIProviderCapabilities struct {
	Chat          bool `json:"chat"`
	Embedding     bool `json:"embedding"`
	Image         bool `json:"image"`
	Transcription bool `json:"transcription"`
	Speech        bool `json:"speech"`
	Reranking     bool `json:"reranking"`
}

// KnownAICapabilities returns the known capability set for a provider type.
// Used as the default and as fallback when live probing is unavailable (Bedrock, Vertex).
func KnownAICapabilities(typ AIProviderType) AIProviderCapabilities {
	switch typ {
	case AIProviderOpenAI:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true, Transcription: true, Speech: true}
	case AIProviderAnthropic:
		return AIProviderCapabilities{Chat: true}
	case AIProviderGoogle:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true}
	case AIProviderMistral:
		return AIProviderCapabilities{Chat: true, Embedding: true}
	case AIProviderCohere:
		return AIProviderCapabilities{Chat: true, Embedding: true, Reranking: true}
	case AIProviderGroq:
		return AIProviderCapabilities{Chat: true, Embedding: true}
	case AIProviderPerplexity:
		return AIProviderCapabilities{Chat: true, Embedding: true}
	case AIProviderDeepSeek:
		return AIProviderCapabilities{Chat: true}
	case AIProviderFireworks:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true}
	case AIProviderTogetherAI:
		return AIProviderCapabilities{Chat: true, Embedding: true, Reranking: true}
	case AIProviderXAI:
		return AIProviderCapabilities{Chat: true}
	case AIProviderAzure:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true, Transcription: true, Speech: true}
	case AIProviderBedrock:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true}
	case AIProviderVertex:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true}
	case AIProviderHuggingFace:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true}
	case AIProviderCerebras:
		return AIProviderCapabilities{Chat: true}
	default:
		return AIProviderCapabilities{Chat: true}
	}
}

// VectorStoreCapabilities describes what operations a vector store supports.
type VectorStoreCapabilities struct {
	CreateIndex   bool `json:"createIndex"`
	DeleteIndex   bool `json:"deleteIndex"`
	ListIndexes   bool `json:"listIndexes"`
	Upsert        bool `json:"upsert"`
	Query         bool `json:"query"`
	UpdateVector  bool `json:"updateVector"`
	DeleteVector  bool `json:"deleteVector"`
	NamedVectors  bool `json:"namedVectors"`  // Qdrant
	SparseVectors bool `json:"sparseVectors"` // Pinecone
}

// DefaultVectorCapabilities returns the standard capability set for most vector stores.
func DefaultVectorCapabilities() VectorStoreCapabilities {
	return VectorStoreCapabilities{
		CreateIndex: true, DeleteIndex: true, ListIndexes: true,
		Upsert: true, Query: true, UpdateVector: true, DeleteVector: true,
	}
}

// StorageCapabilities describes what domains a storage adapter supports.
type StorageCapabilities struct {
	Memory        bool `json:"memory"`
	Workflows     bool `json:"workflows"`
	Scores        bool `json:"scores"`
	Observability bool `json:"observability"`
	Agents        bool `json:"agents"`
	Datasets      bool `json:"datasets"`
	Blobs         bool `json:"blobs"`
}

// DefaultStorageCapabilities returns the standard capability set for most storage adapters.
func DefaultStorageCapabilities() StorageCapabilities {
	return StorageCapabilities{
		Memory: true, Workflows: true, Scores: true,
		Observability: true, Agents: true, Datasets: true,
	}
}
