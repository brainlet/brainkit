// `extractReasoningMiddleware({tagName})` pulls content wrapped in
// the configured XML-like tag out of the model's final text and
// exposes it separately as `reasoning`. Designed for models that
// emit chain-of-thought in `<think>...</think>` blocks
// (DeepSeek-R1, friendli, etc). We exercise the parser by asking
// gpt-4o-mini to produce the shape explicitly.
import { generateText, wrapLanguageModel, extractReasoningMiddleware } from "ai";
import { model, output } from "kit";

const reasoningModel = wrapLanguageModel({
  model: model("openai", "gpt-4o-mini"),
  middleware: extractReasoningMiddleware({ tagName: "think" }),
});

const result = await generateText({
  model: reasoningModel,
  prompt:
    "Wrap your chain of thought in <think>...</think> tags, then give the answer outside the tags.\n" +
    "Question: What is 12 * 7? Keep the think block short.",
});

const reasoning = (result as any).reasoning
  ? (Array.isArray((result as any).reasoning)
      ? (result as any).reasoning.map((r: any) => r.text || "").join("")
      : String((result as any).reasoning))
  : "";
const text = result.text || "";

output({
  hasText: text.length > 0,
  textOmitsThinkTag: !text.includes("<think>") && !text.includes("</think>"),
  hasReasoning: reasoning.length > 0,
  answerMentions84: text.includes("84"),
});
