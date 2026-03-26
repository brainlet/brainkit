// Test: model resolution works for different providers
// Only tests provider that has an API key set
import { model, output } from "kit";
import { generateText } from "ai";

const results: Record<string, any> = {};

// OpenAI (always available in test)
try {
  const r = await generateText({
    model: model("openai", "gpt-4o-mini"),
    prompt: "Say 'ok'",
    maxTokens: 5,
  });
  results.openai = { ok: true, text: r.text.substring(0, 20) };
} catch (e: any) {
  results.openai = { ok: false, error: e.message.substring(0, 100) };
}

// Anthropic (only if key set)
if (process.env.ANTHROPIC_API_KEY) {
  try {
    const r = await generateText({
      model: model("anthropic", "claude-sonnet-4-20250514"),
      prompt: "Say 'ok'",
      maxTokens: 5,
    });
    results.anthropic = { ok: true, text: r.text.substring(0, 20) };
  } catch (e: any) {
    results.anthropic = { ok: false, error: e.message.substring(0, 100) };
  }
}

output({
  openaiWorks: results.openai?.ok === true,
  hasAnthropic: !!results.anthropic,
  providersTested: Object.keys(results).length,
});
