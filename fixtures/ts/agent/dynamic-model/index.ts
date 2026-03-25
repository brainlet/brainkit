// Test: dynamic model resolver — model is a function that reads from RequestContext
import { Agent, RequestContext } from "agent";
import { model, output } from "kit";

const a = new Agent({
  name: "fixture",
  model: ({ requestContext }) => {
    const modelId = requestContext.get("model");
    if (modelId) {
      const parts = modelId.split("/");
      return model(parts[0], parts[1]);
    }
    return model("openai", "gpt-4o-mini");
  },
  instructions: "Reply with EXACTLY: DYNAMIC_MODEL_OK",
});

const ctx = new RequestContext([["model", "openai/gpt-4o-mini"]]);
const result = await a.generate("test", { requestContext: ctx });

output({
  text: result.text,
  works: result.text.includes("DYNAMIC_MODEL_OK"),
});
