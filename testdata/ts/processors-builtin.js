// Test: Built-in processors — pure logic (no LLM needed)
// Verifies: UnicodeNormalizer, TokenLimiterProcessor, ToolCallFilter are importable and constructible
import { agent, createTool, z, processors, output } from "kit";

const results = {};

// 1. UnicodeNormalizer — pure logic, no external deps
try {
  const norm = new processors.UnicodeNormalizer({
    stripControlChars: true,
    collapseWhitespace: true,
    trim: true,
  });
  results.unicodeNormalizer = norm ? "ok" : "null";
  results.unicodeNormalizerId = norm.id || "no id";
} catch(e) {
  results.unicodeNormalizer = "error: " + e.message;
}

// 2. TokenLimiterProcessor — uses js-tiktoken
try {
  const limiter = new processors.TokenLimiterProcessor(4096);
  results.tokenLimiter = limiter ? "ok" : "null";
} catch(e) {
  results.tokenLimiter = "error: " + e.message;
}

// 3. ToolCallFilter — pure logic
try {
  const filter = new processors.ToolCallFilter({ exclude: ["dangerous_tool"] });
  results.toolCallFilter = filter ? "ok" : "null";
} catch(e) {
  results.toolCallFilter = "error: " + e.message;
}

// 4. BatchPartsProcessor — pure logic
try {
  const batcher = new processors.BatchPartsProcessor({ batchSize: 5 });
  results.batchParts = batcher ? "ok" : "null";
} catch(e) {
  results.batchParts = "error: " + e.message;
}

// 5. Test UnicodeNormalizer as input processor on an agent
try {
  const norm = new processors.UnicodeNormalizer({ collapseWhitespace: true, trim: true });
  const a = agent({
    model: "openai/gpt-4o-mini",
    instructions: "Echo back what the user says, exactly.",
    inputProcessors: [norm],
  });

  const r = await a.generate("Hello    world!", { modelSettings: { temperature: 0 } });
  results.agentWithProcessor = r.text.length > 0 ? "ok" : "empty";
} catch(e) {
  results.agentWithProcessor = "error: " + e.message;
}

// 6. Verify all processors are available
const available = [];
for (const name of ["ModerationProcessor", "PromptInjectionDetector", "PIIDetector", "SystemPromptScrubber",
  "UnicodeNormalizer", "LanguageDetector", "TokenLimiterProcessor", "BatchPartsProcessor",
  "StructuredOutputProcessor", "ToolCallFilter", "ToolSearchProcessor"]) {
  if (processors[name]) available.push(name);
}
results.availableCount = available.length;
results.availableList = available.join(",");

output(results);
