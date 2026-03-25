import { generateText, z } from "ai";
import { createTool } from "agent";
import { model, kit, output } from "kit";
let callCount = 0;
const weatherTool = createTool({
  id: "getWeather",
  description: "Get weather for a city",
  inputSchema: z.object({ city: z.string() }),
  execute: async ({ city }) => { callCount++; return { city, temp: 22, conditions: "sunny" }; },
});
kit.register("tool", "getWeather_ai", weatherTool);
const result = await generateText({
  model: model("openai", "gpt-4o-mini"),
  tools: { getWeather: weatherTool },
  maxSteps: 5,
  prompt: "What's the weather in Paris? Use the getWeather tool.",
});
output({ text: result.text, toolCallsMade: callCount, multiStep: result.steps.length > 1 });
