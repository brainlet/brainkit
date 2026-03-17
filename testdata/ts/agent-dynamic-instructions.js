// Test: dynamic instructions resolver — instructions computed per-request
import { agent, RequestContext, output } from "kit";

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: ({ requestContext }) => {
    const word = requestContext.get("keyword") || "DEFAULT";
    return "Reply with EXACTLY: " + word;
  },
});

// Call 1: keyword "ALPHA"
const ctx1 = new RequestContext([["keyword", "ALPHA"]]);
const r1 = await a.generate("say it", { requestContext: ctx1 });

// Call 2: keyword "BETA"
const ctx2 = new RequestContext([["keyword", "BETA"]]);
const r2 = await a.generate("say it", { requestContext: ctx2 });

output({
  text1: r1.text,
  text2: r2.text,
  hasAlpha: r1.text.includes("ALPHA"),
  hasBeta: r2.text.includes("BETA"),
  different: r1.text !== r2.text,
});
