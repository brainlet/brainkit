import { embedMany } from "ai";
import { embeddingModel, output } from "kit";
const result = await embedMany({
  model: embeddingModel("openai", "text-embedding-3-small"),
  values: ["hello world", "goodbye moon", "brainkit rocks"],
});
output({ count: result.embeddings.length, allVectors: result.embeddings.every((e: number[]) => Array.isArray(e) && e.length > 0), dimensions: result.embeddings[0]?.length || 0, hasUsage: result.usage.tokens > 0 });
