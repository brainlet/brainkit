# AI SDK v5 — API Reference (brainkit runtime)

Source of truth: `internal/engine/runtime/ai.d.ts`. The `"ai"` module inside `.ts` deployments re-exports the real Vercel AI SDK v5 — no wrapping, no shim. This doc is the ingestion-ready reference for what the embedded surface exposes.

```typescript
import {
    generateText, streamText, generateObject, streamObject,
    embed, embedMany, tool, jsonSchema,
    defaultSettingsMiddleware, extractReasoningMiddleware, wrapLanguageModel,
    z,
} from "ai";
import { model, embeddingModel } from "kit";
```

`model("openai", "gpt-4o-mini")` returns an opaque `LanguageModel`. `embeddingModel("openai", "text-embedding-3-small")` returns an `EmbeddingModel`. Both are consumed by the functions below — never constructed manually from `.ts`.

---

## CallSettings

Shared fields extended by `GenerateTextParams`, `StreamTextParams`, `GenerateObjectParams`, `StreamObjectParams`.

```typescript
interface CallSettings {
    maxOutputTokens?: number;   // v5 rename — NOT maxTokens
    temperature?: number;
    topP?: number;
    topK?: number;
    presencePenalty?: number;   // -1..1
    frequencyPenalty?: number;  // -1..1
    stopSequences?: string[];
    seed?: number;
    maxRetries?: number;        // default 2
    abortSignal?: AbortSignal;
    timeout?: number | { totalMs?: number; stepMs?: number; chunkMs?: number };
    headers?: Record<string, string | undefined>;
}
```

Note: `maxTokens` is rejected — use `maxOutputTokens`. `timeout` can be a number or a granular object.

---

## Usage

```typescript
interface Usage {
    inputTokens?: number;        // v5
    outputTokens?: number;       // v5
    totalTokens?: number;
    reasoningTokens?: number;    // for reasoning-capable models
    /** @deprecated */ promptTokens?: number;      // v4 alias
    /** @deprecated */ completionTokens?: number;  // v4 alias
}
```

AI SDK v5 renamed `promptTokens`→`inputTokens` and `completionTokens`→`outputTokens`. The v4 names still appear as deprecated aliases for Mastra interop; prefer v5. Mastra's `AgentResult.usage` retains v4 shape — do not conflate.

---

## Shared types

```typescript
type FinishReason = "stop" | "length" | "content-filter" | "tool-calls" | "error" | "other";

interface ResponseMeta {
    id: string;
    modelId: string;
    timestamp: Date;
    headers?: Record<string, string>;
}

interface ToolCall {
    toolCallId: string;
    toolName: string;
    args: Record<string, unknown>;
}

interface ToolResult {
    toolCallId: string;
    toolName: string;
    args: Record<string, unknown>;
    result: unknown;
}

interface StepResult {
    text: string;
    reasoning?: string;
    toolCalls: ToolCall[];
    toolResults: ToolResult[];
    finishReason: FinishReason;
    usage: Usage;
    stepType: "initial" | "tool-result" | "continue";
    isContinued: boolean;
}

interface Source {
    id: string;
    url?: string;
    title?: string;
    sourceType?: string;
    providerMetadata?: ProviderMetadata;
}

type ContentPart =
    | { type: "text"; text: string }
    | { type: "image"; image: string | Uint8Array; mimeType?: string }
    | { type: "tool-call"; toolCallId: string; toolName: string; args: Record<string, unknown> }
    | { type: "tool-result"; toolCallId: string; toolName: string; result: unknown };

type MessageContent = string | ContentPart[];

interface GeneratedFile { data: Uint8Array; mimeType: string; }
interface Warning       { type: string; message: string; }
type ProviderMetadata  = Record<string, Record<string, unknown>>;
type ProviderOptions   = Record<string, Record<string, unknown>>;

interface LanguageModel  { readonly __brand: "LanguageModel"; }
interface EmbeddingModel { readonly __brand: "EmbeddingModel"; }
```

---

## generateText

```typescript
interface GenerateTextParams extends CallSettings {
    model: LanguageModel;
    prompt?: string;
    system?: string;
    messages?: Array<{
        role: "system" | "user" | "assistant" | "tool";
        content: MessageContent;
    }>;
    tools?: Record<string, ToolDefinition>;
    toolChoice?: "auto" | "none" | "required" | { type: "tool"; toolName: string };
    activeTools?: string[];                    // per-step tool allow-list
    stopWhen?: any;                            // default stepCountIs(1); replaces maxSteps
    providerOptions?: ProviderOptions;
    output?: ZodType | any;                    // structured output spec
    prepareStep?: (ctx: {
        model: LanguageModel;
        steps: StepResult[];
        stepNumber: number;
    }) => any;
    onStepFinish?: (event: StepResult) => void | Promise<void>;
    onFinish?: (event: GenerateTextResult) => void | Promise<void>;
    /** @deprecated Use stopWhen. */
    maxSteps?: number;
}

interface GenerateTextResult {
    readonly text: string;
    readonly reasoning: Array<{ type: string; text?: string }>;
    readonly reasoningText: string | undefined;
    readonly toolCalls: ToolCall[];
    readonly toolResults: ToolResult[];
    readonly finishReason: FinishReason;
    readonly usage: Usage;          // last step only
    readonly totalUsage: Usage;     // aggregate across steps
    readonly steps: StepResult[];
    readonly response: ResponseMeta & { messages: any[]; body?: unknown };
    readonly files: GeneratedFile[];
    readonly sources: Source[];
    readonly warnings: Warning[] | undefined;
    readonly providerMetadata?: ProviderMetadata;
    readonly output: unknown;       // populated when params.output set
}

function generateText(params: GenerateTextParams): Promise<GenerateTextResult>;
```

Multi-step agent loops: set `stopWhen` (e.g. `stepCountIs(5)`); do not use `maxSteps`. `response.messages` contains the full assembled conversation for downstream reuse.

---

## streamText

```typescript
interface StreamTextParams extends GenerateTextParams {
    onChunk?: (event: { chunk: StreamPart }) => void;
    onError?: (event: { error: unknown }) => void;
}

type StreamPart =
    | { type: "text-delta";   textDelta: string }
    | { type: "reasoning";    textDelta: string }
    | { type: "tool-call";    toolCallId: string; toolName: string; args: Record<string, unknown> }
    | { type: "tool-result";  toolCallId: string; toolName: string; result: unknown }
    | { type: "step-finish";  finishReason: FinishReason; usage: Usage }
    | { type: "finish";       finishReason: FinishReason; usage: Usage }
    | { type: "error";        error: unknown };

interface StreamTextResult {
    textStream:    AsyncIterable<string>;
    fullStream:    AsyncIterable<StreamPart>;
    text:          Promise<string>;
    reasoning:     Promise<string | undefined>;
    usage:         Promise<Usage>;
    finishReason:  Promise<FinishReason>;
    response:      Promise<ResponseMeta>;
    toolCalls:     Promise<ToolCall[]>;
    toolResults:   Promise<ToolResult[]>;
    steps:         Promise<StepResult[]>;
    sources:       Promise<Source[]>;
}

function streamText(params: StreamTextParams): StreamTextResult;
```

Note: `streamText` returns the result synchronously — not a `Promise`. Iterate `textStream` for user-visible deltas; iterate `fullStream` when you need tool/step events. Use `msg.stream.text(delta)` to forward chunks to the brainkit bus.

---

## generateObject

```typescript
interface GenerateObjectParams extends CallSettings {
    model: LanguageModel;
    prompt?: string;
    system?: string;
    messages?: Array<{
        role: "system" | "user" | "assistant" | "tool";
        content: MessageContent;
    }>;
    schema?: ZodType;                          // Zod (or jsonSchema()-wrapped JSON Schema)
    schemaName?: string;
    schemaDescription?: string;
    mode?: "auto" | "json" | "tool";           // strategy
    output?: "object" | "array" | "enum" | "no-schema";
    enum?: string[];                           // required when output: "enum"
    providerOptions?: ProviderOptions;
}

interface GenerateObjectResult<T = unknown> {
    object: T;
    finishReason: FinishReason;
    usage: Usage;
    response: ResponseMeta;
    warnings: Warning[];
    providerMetadata?: ProviderMetadata;
}

function generateObject<T = unknown>(params: GenerateObjectParams): Promise<GenerateObjectResult<T>>;
```

`output: "array"` returns `object: T[]`. `output: "enum"` returns `object: string` constrained to `params.enum`. `output: "no-schema"` returns unvalidated `object: unknown`.

---

## streamObject

```typescript
interface StreamObjectParams extends GenerateObjectParams {
    onError?:  (event: { error: unknown }) => void;
    onFinish?: (event: { object: unknown; usage: Usage }) => void;
}

interface StreamObjectResult<T = unknown> {
    partialObjectStream: AsyncIterable<Partial<T>>;
    elementStream:       AsyncIterable<T>;      // populated when output: "array"
    object:              Promise<T>;
    usage:               Promise<Usage>;
    response:            Promise<ResponseMeta>;
}

function streamObject<T = unknown>(params: StreamObjectParams): StreamObjectResult<T>;
```

Returns synchronously (not a `Promise`). Use `partialObjectStream` for progressive UI; use `elementStream` for streaming array outputs element-by-element.

---

## embed / embedMany

```typescript
interface EmbedParams {
    model: EmbeddingModel;
    value: string;
    maxRetries?: number;       // default 2
    abortSignal?: AbortSignal;
    headers?: Record<string, string>;
    providerOptions?: ProviderOptions;
}

interface EmbedResult {
    embedding: number[];
    usage: { tokens: number };
}

function embed(params: EmbedParams): Promise<EmbedResult>;

interface EmbedManyParams {
    model: EmbeddingModel;
    values: string[];
    maxParallelCalls?: number; // default Infinity
    maxRetries?: number;
    abortSignal?: AbortSignal;
    headers?: Record<string, string>;
    providerOptions?: ProviderOptions;
}

interface EmbedManyResult {
    embeddings: number[][];    // index-aligned with values
    usage: { tokens: number };
}

function embedMany(params: EmbedManyParams): Promise<EmbedManyResult>;
```

`usage.tokens` is the only counter reported (no input/output split for embeddings).

---

## Tools: `tool()` + `jsonSchema()`

```typescript
interface ToolDefinition {
    description?: string;
    parameters: ZodType;
    execute?: (args: Record<string, unknown>, options?: { abortSignal?: AbortSignal }) => Promise<unknown>;
}

function tool<T = Record<string, unknown>>(definition: {
    description?: string;
    parameters: ZodType;
    execute?: (args: T, options?: { abortSignal?: AbortSignal }) => Promise<unknown>;
}): ToolDefinition;

function jsonSchema(schema: Record<string, unknown>): ZodType;
```

`tool()` is the AI SDK helper — a plain object describing a callable. For Mastra agent usage, use Mastra's `createTool` instead (see `mastra.md`). `jsonSchema()` wraps a raw JSON Schema dictionary so it is accepted wherever `ZodType` is expected.

```typescript
const lookup = tool({
    description: "Look up a user by id",
    parameters: z.object({ id: z.string() }),
    execute: async ({ id }) => ({ name: "Alice", id }),
});

await generateText({
    model: model("openai", "gpt-4o-mini"),
    prompt: "Find user alice",
    tools: { lookup },
    stopWhen: /* stepCountIs(3) */ undefined,  // rely on default or use stopWhen helpers
});
```

---

## Middleware

```typescript
interface LanguageModelMiddleware { readonly __brand: "LanguageModelMiddleware"; }

function defaultSettingsMiddleware(settings: { settings: Partial<CallSettings> }): LanguageModelMiddleware;
function extractReasoningMiddleware(options?: { tagName?: string; separator?: string }): LanguageModelMiddleware;

function wrapLanguageModel(options: {
    model: LanguageModel;
    middleware: LanguageModelMiddleware | LanguageModelMiddleware[];
}): LanguageModel;
```

`defaultSettingsMiddleware({ settings: { temperature: 0.2, maxOutputTokens: 512 } })` — forces defaults on every call.

`extractReasoningMiddleware({ tagName: "thinking" })` — strips `<thinking>…</thinking>` from `text` and moves it to `reasoning`/`reasoningText`.

```typescript
const base = model("openai", "gpt-4o-mini");
const tuned = wrapLanguageModel({
    model: base,
    middleware: [
        defaultSettingsMiddleware({ settings: { temperature: 0.2 } }),
        extractReasoningMiddleware({ tagName: "thinking" }),
    ],
});
```

---

## Zod (`z`)

```typescript
const z: Zod;

interface Zod {
    string(): ZodType;
    number(): ZodType;
    boolean(): ZodType;
    date(): ZodType;
    any(): ZodType;
    unknown(): ZodType;
    void(): ZodType;
    null(): ZodType;
    undefined(): ZodType;
    never(): ZodType;
    literal(value: string | number | boolean): ZodType;
    enum<T extends [string, ...string[]]>(values: T): ZodType;
    array(schema: ZodType): ZodType;
    object(shape: Record<string, ZodType>): ZodType;
    record(keyType: ZodType, valueType: ZodType): ZodType;
    tuple(items: ZodType[]): ZodType;
    union(types: ZodType[]): ZodType;
    intersection(a: ZodType, b: ZodType): ZodType;
    optional(schema: ZodType): ZodType;
    nullable(schema: ZodType): ZodType;
}

interface ZodType {
    optional(): ZodType;
    nullable(): ZodType;
    default(value: unknown): ZodType;
    describe(description: string): ZodType;   // surfaced in tool JSON schema
    array(): ZodType;
    or(other: ZodType): ZodType;
    and(other: ZodType): ZodType;
    transform(fn: (val: unknown) => unknown): ZodType;
    refine(fn: (val: unknown) => boolean, message?: string): ZodType;
    parse(value: unknown): unknown;
    safeParse(value: unknown): { success: boolean; data?: unknown; error?: ZodError };
}

interface ZodError {
    issues: Array<{ message: string; path: (string | number)[] }>;
    message: string;
}
```

The `z` symbol is the same instance exported from both `"ai"` and `"agent"` modules — schemas defined in one place can be reused in the other.

`.describe("…")` propagates to the JSON Schema emitted for tool parameters — use it to give the model useful argument hints.

---

## Full example (generation + tool + streaming)

```typescript
import { generateText, streamText, tool, z } from "ai";
import { model } from "kit";

const lookup = tool({
    description: "Look up user by id",
    parameters: z.object({ id: z.string().describe("user id") }),
    execute: async ({ id }) => ({ name: "alice", id }),
});

// One-shot with tool loop
const r = await generateText({
    model: model("openai", "gpt-4o-mini"),
    system: "You look up users using the provided tool.",
    prompt: "Find user alice",
    tools: { lookup },
    maxOutputTokens: 256,
    temperature: 0,
    // stopWhen: stepCountIs(3),  // if/when a stopWhen helper is in scope
});
console.log(r.text, r.totalUsage.totalTokens);

// Streaming (bridge deltas to the brainkit bus)
bus.on("chat", async (msg) => {
    const s = streamText({
        model: model("openai", "gpt-4o-mini"),
        prompt: String(msg.payload?.prompt || ""),
    });
    for await (const delta of s.textStream) {
        msg.stream.text(delta);
    }
    msg.stream.end({ usage: await s.usage });
});
```

---

## Usage shape mismatch — reminder

- AI SDK v5 (`"ai"`): `usage.inputTokens` / `usage.outputTokens` / `usage.totalTokens`.
- Mastra (`"agent"`) `AgentResult.usage`: v4 names `promptTokens` / `completionTokens` / `totalTokens`.

When forwarding totals across the boundary, normalise once. See `examples/agent-spawner/main.go` for a spawned-agent handler that coerces both shapes:

```javascript
const u = r.usage || {};
msg.reply({
    text: r.text,
    usage: {
        promptTokens:     u.inputTokens     || u.promptTokens     || 0,
        completionTokens: u.outputTokens    || u.completionTokens || 0,
        totalTokens:      u.totalTokens     || 0,
    },
});
```
