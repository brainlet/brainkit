# Processors in Brainkit

Processors are middleware that transform agent input/output. They run before the LLM (input processors) and after (output processors). Brainkit includes 11 built-in processors for security, data transformation, token management, and tool control.

---

## Quick Start

```ts
import { agent, processors } from "kit";

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are helpful.",
  inputProcessors: [
    new processors.UnicodeNormalizer({ collapseWhitespace: true }),
    new processors.TokenLimiterProcessor(4096),
  ],
  outputProcessors: [
    new processors.ModerationProcessor({ model: "openai/gpt-4o-mini", strategy: "block" }),
  ],
});
```

---

## Available Processors

### Security & Safety

| Processor | LLM Required | Purpose |
|-----------|-------------|---------|
| `ModerationProcessor` | Yes | Content safety (hate, violence, sexual, self-harm) |
| `PromptInjectionDetector` | Yes | Detect injection, jailbreak, exfiltration attempts |
| `PIIDetector` | Yes | Detect and redact PII (email, phone, SSN, API keys) |
| `SystemPromptScrubber` | Yes | Detect system prompt leakage in output |

### Data Transformation

| Processor | LLM Required | Purpose |
|-----------|-------------|---------|
| `UnicodeNormalizer` | No | NFKC normalization, strip control chars, collapse whitespace |
| `LanguageDetector` | Yes | Detect language, optionally translate |

### Stream & Token

| Processor | LLM Required | Purpose |
|-----------|-------------|---------|
| `TokenLimiterProcessor` | No | Limit input/output tokens (js-tiktoken) |
| `BatchPartsProcessor` | No | Batch stream chunks to reduce overhead |
| `StructuredOutputProcessor` | Yes | Transform output to match a Zod schema |

### Tool Management

| Processor | LLM Required | Purpose |
|-----------|-------------|---------|
| `ToolCallFilter` | No | Remove tool calls from message history |
| `ToolSearchProcessor` | No | BM25 search over 100+ tools, load on demand |

---

## Security Processors

### ModerationProcessor

```ts
new processors.ModerationProcessor({
  model: "openai/gpt-4o-mini",       // any LLM, not OpenAI-specific
  threshold: 0.5,                      // confidence threshold (0-1)
  strategy: "block",                   // "block" | "warn" | "filter"
  categories: ["hate", "violence"],    // default: all 11 OpenAI categories
});
```

### PromptInjectionDetector

```ts
new processors.PromptInjectionDetector({
  model: "openai/gpt-4o-mini",
  threshold: 0.7,                      // higher = less sensitive
  strategy: "block",                   // "block" | "warn" | "filter" | "rewrite"
  // "rewrite" neutralizes injection while preserving user intent
});
```

Detection types: injection, jailbreak, tool-exfiltration, data-exfiltration, system-override, role-manipulation.

### PIIDetector

```ts
new processors.PIIDetector({
  model: "openai/gpt-4o-mini",
  strategy: "redact",                  // "block" | "warn" | "filter" | "redact"
  redactionMethod: "mask",             // "mask" (***) | "hash" (SHA256) | "remove" | "placeholder" ([EMAIL])
  preserveFormat: true,                // ***-**-1234 for phone numbers
});
```

PII types: email, phone, credit-card, ssn, api-key, ip-address, name, address, date-of-birth, url, uuid, crypto-wallet, iban.

---

## Pure Logic Processors

### UnicodeNormalizer

```ts
new processors.UnicodeNormalizer({
  stripControlChars: true,     // remove control characters
  preserveEmojis: true,        // keep emojis (default)
  collapseWhitespace: true,    // collapse consecutive whitespace
  trim: true,                  // trim leading/trailing
});
```

Prevents homograph attacks (Cyrillic `а` vs Latin `a`). No external dependencies.

### TokenLimiterProcessor

```ts
// Simple: just a limit
new processors.TokenLimiterProcessor(4096);

// Full options
new processors.TokenLimiterProcessor({
  limit: 4096,
  strategy: "truncate",        // "truncate" | "abort"
});
```

Uses `js-tiktoken` (GPT-4o encoding). System messages are never removed. Prioritizes recent messages.

### ToolCallFilter

```ts
// Remove all tool calls from history
new processors.ToolCallFilter();

// Remove specific tools only
new processors.ToolCallFilter({ exclude: ["dangerous_tool", "internal_tool"] });
```

### ToolSearchProcessor

For agents with 100+ tools — provides `search_tools` and `load_tool` meta-tools instead of dumping all tool descriptions into context.

```ts
new processors.ToolSearchProcessor({
  tools: allMyTools,           // Record<string, Tool>
  search: { topK: 5 },        // how many tools to return per search
  ttl: 3600000,                // cache lifetime (1 hour default)
});
```

---

## Per-Call Processors

Override agent-level processors for a specific call:

```ts
await a.generate("Translate this to French", {
  inputProcessors: [new processors.LanguageDetector({
    model: "openai/gpt-4o-mini",
    targetLanguages: ["English"],
    strategy: "translate",
  })],
});
```

---

## Processor Lifecycle

```
User message
  │
  ├─ processInput()        ← Input processors run here (filtering, normalization)
  │
  ├─ processInputStep()    ← Per agentic-loop step (tool search, context injection)
  │
  ├─ LLM generates response
  │
  ├─ processOutputStream() ← Stream chunk handling (batching, token limiting)
  │
  ├─ processOutputStep()   ← Post-LLM per-step (moderation, PII detection)
  │
  └─ processOutputResult() ← Final output (structured output extraction)
```

Any processor can call `abort()` to stop execution and trigger a tripwire.

---

## Testing

| Test | What it proves |
|------|---------------|
| `TestProcessorsBuiltin` | All 11 processors importable, constructible. UnicodeNormalizer works as agent input processor. |
