// AI SDK `tool()` factory + `stepCountIs()` and `hasToolCall()`
// stop conditions. In multi-step runs the model can chain tool
// calls; stopWhen is how you bound the loop. `stepCountIs(n)` caps
// total steps; `hasToolCall("name")` stops after a specific tool
// fires. Returns an array — the first condition to match wins.
import { generateText, tool, stepCountIs, hasToolCall, z } from "ai";
import { model, output } from "kit";

const weather = tool({
  description: "Gets current weather for a city",
  inputSchema: z.object({ city: z.string() }),
  execute: async ({ city }) => ({
    city,
    tempC: 18,
    conditions: "clear",
  }),
});

const result = await generateText({
  model: model("openai", "gpt-4o-mini"),
  stopWhen: [stepCountIs(5), hasToolCall("weather")],
  tools: { weather },
  prompt: "What's the weather in Paris? Use the weather tool.",
});

const toolResults = (result.steps || []).flatMap(
  (s: any) => s.toolResults || [],
);
const weatherCalled = (result.steps || []).some((s: any) =>
  (s.toolCalls || []).some((c: any) => c.toolName === "weather"),
);

output({
  finishReason: result.finishReason,
  stepCount: (result.steps || []).length,
  weatherCalled,
  toolResultCount: toolResults.length,
  respectsMaxSteps: (result.steps || []).length <= 5,
});
