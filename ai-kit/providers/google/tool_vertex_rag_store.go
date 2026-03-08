// Ported from: packages/google/src/tool/vertex-rag-store.ts
package google

// VertexRagStoreToolID is the tool ID for Vertex RAG Store.
const VertexRagStoreToolID = "google.vertex_rag_store"

// VertexRagStoreToolArgs contains the arguments for the Vertex RAG Store tool.
type VertexRagStoreToolArgs struct {
	// RagCorpus is the resource name, e.g. projects/{project}/locations/{location}/ragCorpora/{rag_corpus}
	RagCorpus string `json:"ragCorpus"`

	// TopK is the number of top contexts to retrieve.
	TopK *int `json:"topK,omitempty"`
}
