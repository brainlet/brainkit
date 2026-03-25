import { output } from "kit";
try {
  const embed = globalThis.__agent_embed;
  output({
    hasRerank: typeof embed.rerank === "function",
    hasCreateVectorQueryTool: typeof embed.createVectorQueryTool === "function",
    hasGraphRAG: typeof embed.GraphRAG === "function",
    hasMDocument: typeof embed.MDocument === "function",
  });
} catch(e: any) { output({ error: e.message }); }
