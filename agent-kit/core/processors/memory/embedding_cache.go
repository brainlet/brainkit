// Ported from: packages/core/src/processors/memory/embedding-cache.ts
package memory

import (
	lrucache "github.com/brainlet/brainkit/lru-cache"
)

const defaultCacheMaxSize = 1000

// globalEmbeddingCache is a process-wide embedding cache shared across all
// SemanticRecall instances.  This ensures embeddings are cached and reused even
// when new processor instances are created.
//
// Cache key format: xxhash hex of "${indexName}:${content}"
// Cache value: embedding vector ([]float64)
var globalEmbeddingCache = lrucache.New[string, []float64](lrucache.Options[string, []float64]{
	Max: defaultCacheMaxSize,
})
