// Test: generateText with temperature 0 — deterministic output
import { generateText } from "ai";
import { model, output } from "kit";

const result1 = await generateText({
  model: model("openai", "gpt-4o-mini"),
  prompt: "What is 2+2? Reply with ONLY the number.",
  temperature: 0,
});

const result2 = await generateText({
  model: model("openai", "gpt-4o-mini"),
  prompt: "What is 2+2? Reply with ONLY the number.",
  temperature: 0,
});

output({
  text1: result1.text.trim(),
  text2: result2.text.trim(),
  deterministic: result1.text.trim() === result2.text.trim(),
  contains4: result1.text.includes("4"),
});
