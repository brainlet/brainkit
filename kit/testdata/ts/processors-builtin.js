// Test: Built-in processors — pure logic (no LLM needed)
// NOTE: processors is a removed API. This fixture documents the old pattern
// and needs rewriting once the new processor pattern is defined.
import { Agent, createTool, z } from "agent";
import { model, output } from "kit";

const results = {};

// The old processors API (UnicodeNormalizer, TokenLimiterProcessor, ToolCallFilter,
// BatchPartsProcessor, etc.) has been removed in the new module split.
// These were available from the old "kit" module as `processors.*`.
//
// TODO: Rewrite this fixture once the new processor pattern is defined in Mastra.
// For now, we test that the agent still works without processors.

try {
  const a = new Agent({
    name: "fixture",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Echo back what the user says, exactly.",
  });

  const r = await a.generate("Hello world!", { modelSettings: { temperature: 0 } });
  results.agentWithoutProcessor = r.text.length > 0 ? "ok" : "empty";
} catch(e) {
  results.agentWithoutProcessor = "error: " + e.message;
}

results.note = "processors API removed — needs rewriting";

output(results);
