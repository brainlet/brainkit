import { streamText } from "ai";
import { model, output } from "kit";
const result = streamText({ model: model("openai", "gpt-4o-mini"), prompt: "Count from 1 to 5" });
const parts: string[] = [];
for await (const part of result.fullStream) { parts.push(part.type); }
const text = await result.text;
output({ partTypes: [...new Set(parts)], hasTextDelta: parts.includes("text-delta"), text: text.substring(0, 100) });
