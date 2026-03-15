// Test: dynamic model resolver — model is a function that reads from RequestContext
import { agent, RequestContext, output } from "brainlet";

const a = agent({
  model: ({ requestContext }) => {
    const modelId = requestContext.get("model");
    return modelId || "openai/gpt-4o-mini";
  },
  instructions: "Reply with EXACTLY: DYNAMIC_MODEL_OK",
});

const ctx = new RequestContext([["model", "openai/gpt-4o-mini"]]);
const result = await a.generate("test", { requestContext: ctx });

output({
  text: result.text,
  works: result.text.includes("DYNAMIC_MODEL_OK"),
});
