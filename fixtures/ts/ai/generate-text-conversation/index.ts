import { generateText } from "ai";
import { model, output } from "kit";
const result = await generateText({
  model: model("openai", "gpt-4o-mini"),
  messages: [
    { role: "user", content: "My name is Alice and I live in Tokyo." },
    { role: "assistant", content: "Nice to meet you, Alice! Tokyo is wonderful." },
    { role: "user", content: "What's my name and where do I live?" },
  ],
});
output({ text: result.text, remembersName: result.text.toLowerCase().includes("alice"), remembersCity: result.text.toLowerCase().includes("tokyo") });
