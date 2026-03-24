# AI SDK Features — Complete Mapping

> Complete inventory of ALL Vercel AI SDK features vs what brainkit exposes at every layer.
> AI SDK source: `/Users/davidroman/Documents/code/clones/ai/packages/ai/src/`
>
> Two bundles exist in brainkit:
> - **ai-embed** (`internal/embed/ai/bundle/entry.mjs`) → `globalThis.__ai_sdk` — standalone Go client
> - **agent-embed** (`internal/embed/agent/bundle/entry.mjs`) → `globalThis.__agent_embed` — Kit runtime (Mastra + AI SDK)

---

## 1. AI SDK Complete Feature Inventory

### 1.1 Core Generation Functions

| Function | SDK Status | Description |
|----------|:----------:|-------------|
| `generateText` | Stable | Text generation with tools, multi-step, structured output |
| `streamText` | Stable | Streaming text with real-time chunks, tools, multi-step |
| `generateObject` | Stable | Generate typed JSON objects from schema |
| `streamObject` | Stable | Stream partial objects as they build |
| `embed` | Stable | Single value → vector embedding |
| `embedMany` | Stable | Batch embed multiple values |

### 1.2 Media Generation Functions

| Function | SDK Status | Description |
|----------|:----------:|-------------|
| `generateImage` | Stable | Image generation from text/image prompts |
| `generateSpeech` | Experimental | Text-to-speech synthesis |
| `experimental_generateVideo` | Experimental | Text/image-to-video generation |
| `experimental_transcribe` | Experimental | Audio-to-text transcription |

### 1.3 Search & Retrieval

| Function | SDK Status | Description |
|----------|:----------:|-------------|
| `rerank` | Stable | Rerank documents by relevance to a query |

### 1.4 Agent System

| Feature | SDK Status | Description |
|---------|:----------:|-------------|
| `ToolLoopAgent` | Stable | Agent that loops tools until stop condition met |
| `Agent` interface | Stable | Standard `.call()` / `.stream()` contract |
| `createAgentUIStream` | Stable | Stream agent execution to UI clients |
| `createAgentUIStreamResponse` | Stable | HTTP SSE response from agent stream |
| `pipeAgentUIStreamToResponse` | Stable | Pipe agent stream to Node ServerResponse |

### 1.5 Middleware

| Middleware | SDK Status | Description |
|-----------|:----------:|-------------|
| `defaultSettingsMiddleware` | Stable | Apply default CallSettings to any model |
| `extractReasoningMiddleware` | Stable | Extract `<thinking>` XML tags from output |
| `extractJsonMiddleware` | Stable | Extract JSON from markdown code fences |
| `simulateStreamingMiddleware` | Stable | Make non-streaming models appear to stream |
| `addToolInputExamplesMiddleware` | Stable | Inject examples into tool parameter schemas |
| `wrapLanguageModel` | Stable | Generic language model wrapper (custom middleware) |
| `wrapEmbeddingModel` | Stable | Generic embedding model wrapper |
| `wrapImageModel` | Stable | Generic image model wrapper |
| `wrapProvider` | Stable | Generic provider wrapper |
| `defaultEmbeddingSettingsMiddleware` | Stable | Default settings for embedding models |

### 1.6 CallSettings (shared across all generate/stream functions)

| Setting | Type | Description |
|---------|------|-------------|
| `maxOutputTokens` | `number` | Max tokens to generate |
| `temperature` | `number` | Sampling temperature |
| `topP` | `number` | Nucleus sampling |
| `topK` | `number` | Top-K sampling |
| `presencePenalty` | `number` | Penalize repeated prompt content |
| `frequencyPenalty` | `number` | Penalize repeated words |
| `stopSequences` | `string[]` | Stop generation at these strings |
| `seed` | `number` | Deterministic sampling |
| `maxRetries` | `number` | Retry failed API calls (default: 2) |
| `abortSignal` | `AbortSignal` | Cancel the call |
| `timeout` | `number \| {totalMs?, stepMs?, chunkMs?}` | Timeout configuration |
| `headers` | `Record<string, string>` | Extra HTTP headers |

### 1.7 generateText Parameters (complete)

| Parameter | Type | Description |
|-----------|------|-------------|
| `model` | `LanguageModel` | The model to use |
| `prompt` | `string` | Simple text prompt |
| `system` | `string` | System message |
| `messages` | `ModelMessage[]` | Conversation messages |
| `tools` | `ToolSet` | Available tools with schema + execute |
| `toolChoice` | `auto\|none\|required\|{type:"tool",toolName}` | Tool selection strategy |
| `activeTools` | `string[]` | Subset of tools available per step |
| `stopWhen` | `StopCondition` | Multi-step stop condition (default: `stepCountIs(1)`) |
| `output` | `Output` | Structured output type (text or object schema) |
| `prepareStep` | `fn` | Dynamic tool/system/settings per step |
| `experimental_repairToolCall` | `fn` | Fix malformed tool calls |
| `experimental_download` | `fn` | Custom URL download handler |
| `experimental_context` | `any` | Context passed to tool calls |
| `experimental_include` | `string[]` | Include extra data in response |
| `providerOptions` | `Record<string, Record<string, any>>` | Per-provider options |
| `onStart` | `callback` | Before any LLM call |
| `onStepStart` | `callback` | Before each step |
| `onStepFinish` | `callback` | After each step |
| `onToolCallStart` | `callback` | Before each tool execution |
| `onToolCallFinish` | `callback` | After each tool execution |
| `onFinish` | `callback` | After all steps complete |

### 1.8 streamText Additional Parameters (beyond generateText)

| Parameter | Type | Description |
|-----------|------|-------------|
| `experimental_transform` | `fn` | Transform stream parts |
| `includeRawChunks` | `boolean` | Include raw provider chunks |
| `onChunk` | `callback` | Per-chunk callback |
| `onError` | `callback` | Error callback |

### 1.9 streamText Return Value

| Property | Type | Description |
|----------|------|-------------|
| `textStream` | `AsyncIterable<string>` | Text delta stream |
| `fullStream` | `AsyncIterable<StreamPart>` | Typed stream (text-delta, tool-call, tool-result, step-finish, error, etc.) |
| `text` | `Promise<string>` | Final complete text |
| `reasoning` | `Promise<string>` | Extracted reasoning |
| `usage` | `Promise<Usage>` | Token usage |
| `finishReason` | `Promise<FinishReason>` | Why generation stopped |
| `response` | `Promise<ResponseMeta>` | Response metadata |
| `toolCalls` | `Promise<ToolCall[]>` | All tool calls |
| `toolResults` | `Promise<ToolResult[]>` | All tool results |
| `steps` | `Promise<StepResult[]>` | All steps |
| `sources` | `Promise<Source[]>` | Source attributions |
| `toUIMessageStreamResponse()` | `fn` | Convert to HTTP SSE response |
| `pipeUIMessageStreamToResponse()` | `fn` | Pipe to Node.js ServerResponse |

### 1.10 generateObject Parameters (complete)

| Parameter | Type | Description |
|-----------|------|-------------|
| `model` | `LanguageModel` | The model to use |
| `prompt`/`system`/`messages` | — | Standard prompt config |
| `schema` | `Schema` | Zod/JSON schema for output |
| `schemaName` | `string` | Optional name for the schema |
| `schemaDescription` | `string` | Optional description |
| `mode` | `"auto"\|"json"\|"tool"` | Generation strategy |
| `output` | `"object"\|"array"\|"enum"\|"no-schema"` | Output type |
| `enum` | `any[]` | Enum values (for `output: "enum"`) |
| `experimental_repairObject` | `fn` | Fix malformed output |
| All CallSettings | — | temperature, maxOutputTokens, etc. |
| `providerOptions` | — | Per-provider options |

### 1.11 streamObject Additional Return Value

| Property | Type | Description |
|----------|------|-------------|
| `partialObjectStream` | `AsyncIterable<Partial<T>>` | Partial objects as they build |
| `elementStream` | `AsyncIterable<T>` | Individual elements (for `output: "array"`) |
| `object` | `Promise<T>` | Final complete object |
| `usage` | `Promise<Usage>` | Token usage |

### 1.12 Registry & Provider System

| Feature | Description |
|---------|-------------|
| `customProvider` | Create provider with model ID mapping |
| `createProviderRegistry` | Registry for resolving "provider:model" strings |
| `NoSuchProviderError` | Error type |

### 1.13 Tool Utilities

| Feature | Description |
|---------|-------------|
| `tool()` | Define a tool (schema + execute + description) |
| `dynamicTool()` | Tool with runtime-changeable definition |
| `jsonSchema()` | Convert JSON Schema → AI SDK Schema |
| `zodSchema()` | Convert Zod → AI SDK Schema |
| `asSchema()` | Type-safe schema wrapper |

### 1.14 Telemetry

| Feature | Description |
|---------|-------------|
| `TelemetrySettings` | OpenTelemetry configuration |
| `experimental_telemetry` | Param on all functions for tracing |

### 1.15 Text Stream Utilities

| Feature | Description |
|---------|-------------|
| `createTextStreamResponse` | Create HTTP Response from text stream |
| `pipeTextStreamToResponse` | Pipe text stream to Node ServerResponse |
| `smoothStream` | Transform to smooth chunky streams |

### 1.16 UI Integration

| Feature | Description |
|---------|-------------|
| `AbstractChat` | Base chat state management class |
| `UIMessage` / `UIMessagePart` | Typed message structure |
| `UIMessageStream` | Streaming protocol (JSON-to-SSE) |
| `createUIMessageStream` | Create a UI message stream |
| `readUIMessageStream` | Read/parse a UI message stream |
| `UIDataPartSchemas` | Schema for custom data parts |
| Chat transports | HTTP, direct, text-stream |
| `convertToModelMessages` | UIMessage → ModelMessage conversion |

### 1.17 Error Types (20+)

`AISDKError`, `APICallError`, `EmptyResponseBodyError`, `InvalidPromptError`, `InvalidResponseDataError`, `JSONParseError`, `LoadAPIKeyError`, `NoContentGeneratedError`, `NoSuchModelError`, `TooManyEmbeddingValuesForCallError`, `TypeValidationError`, `UnsupportedFunctionalityError`, `InvalidArgumentError`, `InvalidToolInputError`, `NoObjectGeneratedError`, `NoImageGeneratedError`, `NoSpeechGeneratedError`, `NoTranscriptGeneratedError`, `NoVideoGeneratedError`, `ToolCallRepairError`, `UIMessageStreamError`, `RetryError`, etc.

### 1.18 ID Generation

| Feature | Description |
|---------|-------------|
| `generateId()` | Generate a unique ID |
| `createIdGenerator()` | Custom generator with prefix + size |

---

## 2. Bundle Availability Matrix

### ai-embed bundle (`__ai_sdk`)

| Export | Available | Notes |
|--------|:---------:|-------|
| `generateText` | Y | |
| `streamText` | Y | |
| `generateObject` | Y | |
| `streamObject` | Y | |
| `embed` | Y | |
| `embedMany` | Y | |
| `tool` | Y | |
| `jsonSchema` | Y | |
| `wrapLanguageModel` | Y | |
| `defaultSettingsMiddleware` | Y | |
| `extractReasoningMiddleware` | Y | |
| `createOpenAI` | Y | |
| `createAnthropic` | Y | |
| `createGoogleGenerativeAI` | Y | |
| `generateImage` | **NO** | Not imported |
| `generateSpeech` | **NO** | Not imported |
| `generateVideo` | **NO** | Not imported |
| `transcribe` | **NO** | Not imported |
| `rerank` | **NO** | Not imported |
| `ToolLoopAgent` | **NO** | Not imported |
| `extractJsonMiddleware` | **NO** | Not imported |
| `simulateStreamingMiddleware` | **NO** | Not imported |
| `addToolInputExamplesMiddleware` | **NO** | Not imported |
| `wrapEmbeddingModel` | **NO** | Not imported |
| `smoothStream` | **NO** | Not imported |
| `customProvider` | **NO** | Not imported |
| `createProviderRegistry` | **NO** | Not imported |
| Extra provider factories (9) | **NO** | Only openai, anthropic, google |

### agent-embed bundle (`__agent_embed`)

| Export | Available | Notes |
|--------|:---------:|-------|
| AI SDK: `generateText`, `streamText`, `generateObject`, `streamObject`, `embed`, `embedMany` | Y | Direct passthrough |
| `ModelRouterEmbeddingModel` | Y | For embedding model resolution |
| `RequestContext` | Y | For dynamic config |
| 12 provider factories | Y | All major providers |
| Mastra: `Agent`, `createTool`, `createWorkflow`, `createStep` | Y | |
| Mastra: `Memory`, `InMemoryStore` | Y | |
| Mastra: Storage backends (5) | Y | LibSQL, Upstash, Postgres, MongoDB, InMemory |
| Mastra: Vector stores (3) | Y | LibSQL, PgVector, MongoDB |
| Mastra: Evals (15 scorers + createScorer + runEvals) | Y | |
| Mastra: Processors (11) | Y | Security, data, stream, tool |
| Mastra: RAG (MDocument, GraphRAG, rerank, tools) | Y | |
| Mastra: Observability | Y | |
| Mastra: Workspace + LocalFilesystem + LocalSandbox | Y | |
| Mastra: Harness + tools | Y | |
| `generateImage` | **NO** | Not imported |
| `generateSpeech` | **NO** | Not imported |
| `generateVideo` | **NO** | Not imported |
| `transcribe` | **NO** | Not imported |
| AI SDK `rerank` | **NO** | Mastra `rerank` is separate |
| `ToolLoopAgent` | **NO** | Only Mastra Agent |
| AI SDK middleware functions | **NO** | Not imported in agent bundle |
| `wrapLanguageModel` | **NO** | Not imported in agent bundle |
| `smoothStream` | **NO** | Not imported |
| `customProvider` | **NO** | |
| `createProviderRegistry` | **NO** | brainkit has its own |

---

## 3. Go Wrapper (`internal/embed/ai`) Coverage

| Method | AI SDK Function | CallSettings | Tools | Middleware | Streaming | ProviderOpts |
|--------|----------------|:------------:|:-----:|:---------:|:---------:|:------------:|
| `GenerateText` | `generateText` | Y | Y (Go cbs) | Y | N/A | Y |
| `StreamText` | `streamText` | Y | **NO** | **NO** | Blocking+OnToken | Y |
| `GenerateObject` | `generateObject` | Y | N/A | **NO** | N/A | Y |
| `StreamObject` | `streamObject` | Y | N/A | **NO** | Blocking+OnPartial | Y |
| `Embed` | `embed` | N/A | N/A | N/A | N/A | **NO** |
| `EmbedMany` | `embedMany` | N/A | N/A | N/A | N/A | **NO** |
| — | `generateImage` | — | — | — | — | — |
| — | `generateSpeech` | — | — | — | — | — |
| — | `generateVideo` | — | — | — | — | — |
| — | `transcribe` | — | — | — | — | — |
| — | `rerank` | — | — | — | — | — |

### Go wrapper feature gaps (per function)

**GenerateText:** Missing `activeTools`, `stopWhen` (multi-step), `output`, `prepareStep`, `repairToolCall`, `timeout`, `headers`, `download`. Missing callbacks: `onStart`, `onStepStart`, `onToolCallStart`, `onToolCallFinish` (only `OnStepFinish` typed but not wired).

**StreamText:** Missing `tools`, `toolChoice`, `middleware`, `fullStream` consumption, `activeTools`, `stopWhen`, `prepareStep`, `onChunk`, `transform`. Only reads `textStream`, blocks until complete.

**GenerateObject:** Missing `output` mode (array/enum/no-schema), `enum` param, `repairObject`.

**StreamObject:** Missing `output` mode, `enum`, `elementStream`, callbacks.

**Embed/EmbedMany:** Missing `providerOptions`, `maxParallelCalls` (embedMany).

---

## 4. kit_runtime.js (`ai.*`) Coverage

### Parameters forwarded to AI SDK

| Method | model | prompt | system | messages | schema | CallSettings | tools | providerOpts | middleware | toolChoice |
|--------|:-----:|:------:|:------:|:--------:|:------:|:------------:|:-----:|:------------:|:---------:|:----------:|
| `ai.generate` | Y | Y | Y | Y | — | **NO** | **NO** | **NO** | **NO** | **NO** |
| `ai.stream` | Y | Y | Y | Y | — | **NO** | **NO** | **NO** | **NO** | **NO** |
| `ai.embed` | Y | — | — | — | — | — | — | **NO** | — | — |
| `ai.embedMany` | Y | — | — | — | — | — | — | **NO** | — | — |
| `ai.generateObject` | Y | Y | Y | Y | Y | **NO** | — | **NO** | **NO** | — |
| `ai.streamObject` | Y | Y | Y | Y | Y | **NO** | — | **NO** | **NO** | — |

### Features that kit_runtime.js provides beyond AI SDK

See complete list in FEATURES.md — agents, tools, workflows, memory, bus, fs, wasm, mcp, registry, evals, processors, RAG, workspace, harness, observability.

---

## 5. Catalog Command Coverage

| Topic | Msg Type | Handler | In Catalog | Tested |
|-------|----------|---------|:----------:|:------:|
| `ai.generate` | `AiGenerateMsg` | `AIDomain.Generate` | Y | Y |
| `ai.embed` | `AiEmbedMsg` | `AIDomain.Embed` | Y | Y |
| `ai.embedMany` | `AiEmbedManyMsg` | `AIDomain.EmbedMany` | Y | Y |
| `ai.generateObject` | `AiGenerateObjectMsg` | `AIDomain.GenerateObject` | Y | Y |
| `ai.stream` | `AiStreamMsg` (exists) | **NONE** | **NO** | **NO** |
| `ai.streamObject` | — | — | **NO** | **NO** |
| `ai.generateImage` | — | — | **NO** | **NO** |
| `ai.generateSpeech` | — | — | **NO** | **NO** |
| `ai.generateVideo` | — | — | **NO** | **NO** |
| `ai.transcribe` | — | — | **NO** | **NO** |
| `ai.rerank` | — | — | **NO** | **NO** |

### Message type field gaps

`AiGenerateMsg`:
```go
Model    string          // Y
Prompt   string          // Y
Messages []AiChatMessage // Y
Tools    []string        // Y (tool names from registry)
Schema   any             // Y (JSON schema for generateObject mode)
// MISSING: system, CallSettings, toolChoice, providerOptions, maxSteps, output
```

`AiStreamMsg`:
```go
Model    string          // Y
Prompt   string          // Y
Messages []AiChatMessage // Y
StreamTo string          // OLD PATTERN — should use replyTo
// MISSING: system, CallSettings, tools, toolChoice, providerOptions
```

---

## 6. Summary — Gap Tiers

### Tier 1: Broken wiring (available but not passed through)

| # | Gap | Where | Impact |
|---|-----|-------|--------|
| 1 | `ai.*` drops ALL CallSettings | kit_runtime.js | Users can't set temperature, maxTokens, etc. from .ts |
| 2 | `ai.*` drops providerOptions | kit_runtime.js | Users can't use provider-specific features from .ts |
| 3 | `ai.generate` drops tools param | kit_runtime.js | .ts `ai.generate({tools})` silently ignores tools |
| 4 | `ai.generate` drops toolChoice | kit_runtime.js | Same |
| 5 | `ai.stream` not in catalog | catalog.go | Go/Plugin/WASM/cross-Kit can't stream at all |
| 6 | `ai.streamObject` not in catalog | catalog.go | Same for object streaming |
| 7 | `AiGenerateMsg` missing system field | messages/ai.go | Bus callers can't set system prompt |
| 8 | `AiGenerateMsg` missing CallSettings | messages/ai.go | Bus callers can't set temperature etc. |
| 9 | `AiStreamMsg` uses `StreamTo` not replyTo | messages/ai.go | Wrong pattern post messaging redesign |

### Tier 2: Not exposed but available in bundles

| # | Gap | Where | Impact |
|---|-----|-------|--------|
| 10 | AI SDK middleware not in agent-embed bundle | bundle entry.mjs | Kit runtime can't use wrapLanguageModel, extractReasoning etc. |
| 11 | Middleware not exposed in kit_runtime.js | kit_runtime.js | .ts code can't apply middleware to models |
| 12 | `StreamText` Go wrapper has no tools | client.go | Go streaming can't use tool-calling |
| 13 | `StreamText` Go wrapper only reads textStream | client.go | Missing fullStream (tool calls, step events, etc.) |
| 14 | Multi-step (stopWhen) not exposed anywhere | All layers | Only single-step generation |
| 15 | `generateObject` output modes (array/enum) not in Go | client.go | Only "object" mode |
| 16 | `smoothStream` not bundled | bundle | No stream smoothing for UI delivery |

### Tier 3: Not bundled, need new imports + handlers

| # | Gap | Where | Impact |
|---|-----|-------|--------|
| 17 | `generateImage` not available | Both bundles | No image generation |
| 18 | `generateSpeech` not available | Both bundles | No TTS |
| 19 | `transcribe` not available | Both bundles | No STT |
| 20 | `generateVideo` not available | Both bundles | No video gen |
| 21 | AI SDK `rerank` not available | Both bundles | Only Mastra rerank |
| 22 | `ToolLoopAgent` not available | Both bundles | Only Mastra Agent |
| 23 | Additional middleware (extractJson, simulateStreaming, addToolInputExamples) | ai-embed bundle | Available but not imported in agent bundle |
| 24 | Missing provider factories in ai-embed | ai-embed bundle | Only 3 of 12 providers |

### Tier 4: UI/Server features (likely N/A for brainkit)

| # | Feature | Reason |
|---|---------|--------|
| 25 | UI message stream / chat transport | brainkit is a runtime, not a web server |
| 26 | `toUIMessageStreamResponse()` | Needs Node.js HTTP — not in QuickJS |
| 27 | `pipeTextStreamToResponse()` | Same |
| 28 | `AbstractChat` | Client-side state management |
