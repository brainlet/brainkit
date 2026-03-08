// Ported from: packages/core/src/processors/memory/index.ts
package memory

// This package will contain memory-related processors.
//
// TODO: Port the following from packages/core/src/processors/memory/:
//
// - MessageHistory (message-history.ts)
//   Type: MessageHistory struct implementing Processor
//   Options: MessageHistoryOptions
//   Purpose: Manages message history truncation and windowing for LLM context.
//
// - WorkingMemory (working-memory.ts)
//   Types: WorkingMemory struct, WorkingMemoryTemplate, WorkingMemoryConfig
//   Purpose: Manages working memory (scratchpad) that persists across conversation turns.
//
// - SemanticRecall (semantic-recall.ts)
//   Types: SemanticRecall struct, SemanticRecallOptions
//   Purpose: Retrieves semantically relevant past messages using vector similarity search.
//
// - globalEmbeddingCache (embedding-cache.ts)
//   Type: Global embedding cache singleton
//   Purpose: Caches embedding results to avoid redundant embedding API calls.
