import { generateText } from "ai";
import { model, output } from "kit";
const result = await generateText({ model: model("openai", "gpt-4o-mini"), prompt: "Say exactly: 'middleware test passed'", maxOutputTokens: 50 });
output({ hasText: result.text.length > 0, finishReason: result.finishReason });
