/**
 * "ai" module — direct re-exports of AI SDK.
 * No wrapping. Users get the real AI SDK functions.
 *
 * @example
 * ```ts
 * import { generateText, streamText, z } from "ai";
 * import { model } from "kit";
 *
 * const result = await generateText({
 *   model: model("openai", "gpt-4o-mini"),
 *   prompt: "Hello",
 *   temperature: 0.7,
 * });
 * ```
 */
declare module "ai" {

  // ── Zod (schema library) ──────────────────────────────────────

  /** Zod schema builder — re-exported from AI SDK. */
  export const z: Zod;

  export interface Zod {
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

  export interface ZodType {
    optional(): ZodType;
    nullable(): ZodType;
    default(value: unknown): ZodType;
    describe(description: string): ZodType;
    array(): ZodType;
    or(other: ZodType): ZodType;
    and(other: ZodType): ZodType;
    transform(fn: (val: unknown) => unknown): ZodType;
    refine(fn: (val: unknown) => boolean, message?: string): ZodType;
    parse(value: unknown): unknown;
    safeParse(value: unknown): { success: boolean; data?: unknown; error?: ZodError };
  }

  export interface ZodError {
    issues: Array<{ message: string; path: (string | number)[] }>;
    message: string;
  }

  // ── CallSettings (shared across all functions) ────────────────

  export interface CallSettings {
    /** Maximum number of tokens to generate (AI SDK v5 name). */
    maxOutputTokens?: number;
    /** Maximum number of tokens to generate (AI SDK v4 name — alias). */
    maxTokens?: number;
    /** Temperature setting. Range depends on provider/model. */
    temperature?: number;
    /** Nucleus sampling (0-1). */
    topP?: number;
    /** Top-K sampling. */
    topK?: number;
    /** Presence penalty (-1 to 1). */
    presencePenalty?: number;
    /** Frequency penalty (-1 to 1). */
    frequencyPenalty?: number;
    /** Stop generation at these sequences. */
    stopSequences?: string[];
    /** Seed for deterministic sampling. */
    seed?: number;
    /** Max retries on failure (default: 2). */
    maxRetries?: number;
    /** Abort signal. */
    abortSignal?: AbortSignal;
    /** Timeout in ms or config object. */
    timeout?: number | { totalMs?: number; stepMs?: number; chunkMs?: number };
    /** Extra HTTP headers. */
    headers?: Record<string, string | undefined>;
  }

  // ── Shared types ──────────────────────────────────────────────

  type FinishReason = "stop" | "length" | "content-filter" | "tool-calls" | "error" | "other";

  export interface Usage {
    /** Input (prompt) tokens. AI SDK v5 name. */
    inputTokens?: number;
    /** Output (completion) tokens. AI SDK v5 name. */
    outputTokens?: number;
    /** Total tokens used. */
    totalTokens?: number;
    /** @deprecated Use inputTokens. Mastra-mapped alias. */
    promptTokens?: number;
    /** @deprecated Use outputTokens. Mastra-mapped alias. */
    completionTokens?: number;
    /** Reasoning tokens (for models that support reasoning). */
    reasoningTokens?: number;
  }

  export interface ResponseMeta {
    id: string;
    modelId: string;
    timestamp: Date;
    headers?: Record<string, string>;
  }

  export interface ToolCall {
    toolCallId: string;
    toolName: string;
    args: Record<string, unknown>;
  }

  export interface ToolResult {
    toolCallId: string;
    toolName: string;
    args: Record<string, unknown>;
    result: unknown;
  }

  export interface StepResult {
    text: string;
    reasoning?: string;
    toolCalls: ToolCall[];
    toolResults: ToolResult[];
    finishReason: FinishReason;
    usage: Usage;
    stepType: "initial" | "tool-result" | "continue";
    isContinued: boolean;
  }

  export interface Source {
    id: string;
    url?: string;
    title?: string;
    sourceType?: string;
    providerMetadata?: ProviderMetadata;
  }

  /** Content part in a multi-modal message. */
  type ContentPart =
    | { type: "text"; text: string }
    | { type: "image"; image: string | Uint8Array; mimeType?: string }
    | { type: "tool-call"; toolCallId: string; toolName: string; args: Record<string, unknown> }
    | { type: "tool-result"; toolCallId: string; toolName: string; result: unknown };

  /** Message content — either a simple string or multi-modal parts. */
  type MessageContent = string | ContentPart[];

  /** A complete LLM message — role + content. */
  export interface Message {
    role: "system" | "user" | "assistant" | "tool";
    content: MessageContent;
  }

  /** Generated file from the model (images, audio, etc). */
  export interface GeneratedFile {
    data: Uint8Array;
    mimeType: string;
  }

  /** Warning from the provider. */
  export interface Warning {
    type: string;
    message: string;
  }

  /** Provider-specific metadata keyed by provider name. */
  type ProviderMetadata = Record<string, Record<string, unknown>>;

  /** Language model instance (opaque — returned by provider factory). */
  export interface LanguageModel {
    /** @internal */ readonly __brand: "LanguageModel";
  }

  /** Embedding model instance (opaque — returned by provider factory). */
  export interface EmbeddingModel {
    /** @internal */ readonly __brand: "EmbeddingModel";
  }

  /**
   * Tool definition for AI SDK. Either the v4 `parameters`
   * field or the v5 `inputSchema` field carries the schema —
   * both are optional because callers sometimes construct
   * tools incrementally.
   */
  export interface ToolDefinition {
    description?: string;
    /** v4 AI SDK name. */
    parameters?: ZodType;
    /** v5 AI SDK name. */
    inputSchema?: ZodType;
    execute?: (args: Record<string, unknown>, options?: { abortSignal?: AbortSignal }) => Promise<unknown>;
  }

  /** Provider options keyed by provider name. */
  type ProviderOptions = Record<string, Record<string, unknown>>;

  // ── generateText ──────────────────────────────────────────────

  export interface GenerateTextParams extends CallSettings {
    /** The language model to use. */
    model: LanguageModel;
    /** Simple text prompt. */
    prompt?: string;
    /** System message. */
    system?: string;
    /** Conversation messages. */
    messages?: Array<{ role: "system" | "user" | "assistant" | "tool"; content: MessageContent }>;
    /** Tool definitions. */
    tools?: Record<string, ToolDefinition>;
    /** Tool selection strategy. */
    toolChoice?: "auto" | "none" | "required" | { type: "tool"; toolName: string };
    /** Limit which tools are active per step. */
    activeTools?: string[];
    /** Stop condition. @default stepCountIs(1) (single step). Use maxSteps-like behavior via stopWhen. */
    stopWhen?: any;
    /** Provider-specific options. */
    providerOptions?: ProviderOptions;
    /** Structured output specification. */
    output?: ZodType | any;
    /** Optional function to configure each step differently. */
    prepareStep?: (ctx: { model: LanguageModel; steps: StepResult[]; stepNumber: number }) => any;
    /** Callback: each step finishes. */
    onStepFinish?: (event: StepResult) => void | Promise<void>;
    /** Callback: all steps done. */
    onFinish?: (event: GenerateTextResult) => void | Promise<void>;
    /** @deprecated Use stopWhen. */
    maxSteps?: number;
  }

  export interface GenerateTextResult {
    /** Generated text from the last step. */
    readonly text: string;
    /** Reasoning output (structured). */
    readonly reasoning: Array<{ type: string; text?: string }>;
    /** Concatenated reasoning text. */
    readonly reasoningText: string | undefined;
    /** Tool calls made in the last step. */
    readonly toolCalls: ToolCall[];
    /** Tool results from the last step. */
    readonly toolResults: ToolResult[];
    /** Why generation stopped. */
    readonly finishReason: FinishReason;
    /** Token usage of the last step. */
    readonly usage: Usage;
    /** Total token usage across all steps. */
    readonly totalUsage: Usage;
    /** All steps in multi-step generation. */
    readonly steps: StepResult[];
    /** Response metadata. */
    readonly response: ResponseMeta & { messages: any[]; body?: unknown };
    /** Generated files (images, audio, etc). */
    readonly files: GeneratedFile[];
    /** Source attributions. */
    readonly sources: Source[];
    /** Warnings from the provider. */
    readonly warnings: Warning[] | undefined;
    /** Provider-specific metadata. */
    readonly providerMetadata?: ProviderMetadata;
    /** Structured output (when output specification used). */
    readonly output: unknown;
  }

  export function generateText(params: GenerateTextParams): Promise<GenerateTextResult>;

  // ── streamText ────────────────────────────────────────────────

  export interface StreamTextParams extends GenerateTextParams {
    /** Callback per stream chunk (v5 name). */
    onChunk?: (event: { chunk: StreamPart }) => void;
    /** Experimental / v4 alias for onChunk. */
    experimental_onChunk?: (event: { chunk: StreamPart }) => void;
    /** Error callback. */
    onError?: (event: { error: unknown }) => void;
    /** Allow provider-specific extensions without killing completion. */
    [key: string]: any;
  }

  export interface StreamTextResult {
    /** Async iterable of text deltas. */
    textStream: AsyncIterable<string>;
    /** Async iterable of typed stream parts (text-delta, tool-call, tool-result, etc). */
    fullStream: AsyncIterable<StreamPart>;
    /** Promise: final complete text. */
    text: Promise<string>;
    /** Promise: extracted reasoning text. */
    reasoning: Promise<string | undefined>;
    /** Promise: token usage. */
    usage: Promise<Usage>;
    /** Promise: finish reason. */
    finishReason: Promise<FinishReason>;
    /** Promise: response metadata. */
    response: Promise<ResponseMeta>;
    /** Promise: all tool calls. */
    toolCalls: Promise<ToolCall[]>;
    /** Promise: all tool results. */
    toolResults: Promise<ToolResult[]>;
    /** Promise: all steps. */
    steps: Promise<StepResult[]>;
    /** Promise: source attributions. */
    sources: Promise<Source[]>;
  }

  type StreamPart =
    | { type: "text-delta"; textDelta: string }
    | { type: "reasoning"; textDelta: string }
    | { type: "tool-call"; toolCallId: string; toolName: string; args: Record<string, unknown> }
    | { type: "tool-result"; toolCallId: string; toolName: string; result: unknown }
    | { type: "step-finish"; finishReason: FinishReason; usage: Usage }
    | { type: "finish"; finishReason: FinishReason; usage: Usage }
    | { type: "error"; error: unknown };

  export function streamText(params: StreamTextParams): StreamTextResult;

  // ── generateObject ────────────────────────────────────────────

  export interface GenerateObjectParams extends CallSettings {
    /** The language model to use. */
    model: LanguageModel;
    /** Simple text prompt. */
    prompt?: string;
    /** System message. */
    system?: string;
    /** Conversation messages. */
    messages?: Array<{ role: "system" | "user" | "assistant" | "tool"; content: MessageContent }>;
    /** Output schema (Zod or JSON Schema). */
    schema?: ZodType;
    /** Optional name for the schema. */
    schemaName?: string;
    /** Optional description for the schema. */
    schemaDescription?: string;
    /** Generation strategy: "auto" | "json" | "tool". */
    mode?: "auto" | "json" | "tool";
    /** Output type. */
    output?: "object" | "array" | "enum" | "no-schema";
    /** Enum values (for output: "enum"). */
    enum?: string[];
    /** Provider-specific options. */
    providerOptions?: ProviderOptions;
  }

  export interface GenerateObjectResult<T = any> {
    /** The generated object. */
    object: T;
    /** Why generation stopped. */
    finishReason: FinishReason;
    /** Token usage. */
    usage: Usage;
    /** Response metadata. */
    response: ResponseMeta;
    /** Warnings. */
    warnings: Warning[];
    /** Provider-specific metadata. */
    providerMetadata?: ProviderMetadata;
  }

  export function generateObject<T = any>(params: GenerateObjectParams): Promise<GenerateObjectResult<T>>;

  // ── streamObject ──────────────────────────────────────────────

  export interface StreamObjectParams extends GenerateObjectParams {
    /** Error callback. */
    onError?: (event: { error: unknown }) => void;
    /** Finish callback. */
    onFinish?: (event: { object: unknown; usage: Usage }) => void;
  }

  export interface StreamObjectResult<T = any> {
    /** Async iterable of partial objects as they build. */
    partialObjectStream: AsyncIterable<Partial<T>>;
    /** Async iterable of elements (for output: "array"). */
    elementStream: AsyncIterable<T>;
    /** Promise: final complete object. */
    object: Promise<T>;
    /** Promise: token usage. */
    usage: Promise<Usage>;
    /** Promise: response metadata. */
    response: Promise<ResponseMeta>;
  }

  export function streamObject<T = any>(params: StreamObjectParams): StreamObjectResult<T>;

  // ── embed ─────────────────────────────────────────────────────

  export interface EmbedParams {
    /** The embedding model to use. */
    model: EmbeddingModel;
    /** The value to embed. */
    value: string;
    /** Max retries (default: 2). */
    maxRetries?: number;
    /** Abort signal. */
    abortSignal?: AbortSignal;
    /** Extra HTTP headers. */
    headers?: Record<string, string>;
    /** Provider-specific options. */
    providerOptions?: ProviderOptions;
  }

  export interface EmbedResult {
    /** The embedding vector. */
    embedding: number[];
    /** Token usage. */
    usage: { tokens: number };
  }

  export function embed(params: EmbedParams): Promise<EmbedResult>;

  // ── embedMany ─────────────────────────────────────────────────

  export interface EmbedManyParams {
    /** The embedding model to use. */
    model: EmbeddingModel;
    /** The values to embed. */
    values: string[];
    /** Max parallel API calls (default: Infinity). */
    maxParallelCalls?: number;
    /** Max retries (default: 2). */
    maxRetries?: number;
    /** Abort signal. */
    abortSignal?: AbortSignal;
    /** Extra HTTP headers. */
    headers?: Record<string, string>;
    /** Provider-specific options. */
    providerOptions?: ProviderOptions;
  }

  export interface EmbedManyResult {
    /** The embedding vectors. */
    embeddings: number[][];
    /** Token usage. */
    usage: { tokens: number };
  }

  export function embedMany(params: EmbedManyParams): Promise<EmbedManyResult>;

  // ── Middleware ─────────────────────────────────────────────────

  /** Apply default CallSettings to a model. */
  export function defaultSettingsMiddleware(settings: { settings: Partial<CallSettings> }): LanguageModelMiddleware;

  /** Extract reasoning from XML tags (e.g., <thinking>). */
  export function extractReasoningMiddleware(options?: { tagName?: string; separator?: string }): LanguageModelMiddleware;

  /** Wrap a language model with middleware. */
  export function wrapLanguageModel(options: { model: LanguageModel; middleware: LanguageModelMiddleware | LanguageModelMiddleware[] }): LanguageModel;

  /** Language model middleware (opaque). */
  export interface LanguageModelMiddleware {
    /** @internal */ readonly __brand: "LanguageModelMiddleware";
  }

  // ── Tool utilities ────────────────────────────────────────────

  /** Define a tool with schema + execute function. */
  export function tool<T = Record<string, unknown>>(definition: {
    description?: string;
    parameters: ZodType;
    execute?: (args: T, options?: { abortSignal?: AbortSignal }) => Promise<unknown>;
  }): ToolDefinition;

  /** Convert a JSON Schema object to an AI SDK schema. */
  export function jsonSchema(schema: Record<string, unknown>): ZodType;

  // ── Stop conditions (v5) ──────────────────────────────────────
  //
  // Composable predicates that halt a multi-step generate/stream
  // loop. Pass them to `stopWhen` on Agent / generateText /
  // streamText calls.

  export type StopCondition = (step: unknown) => boolean | Promise<boolean>;

  /** Stop once the loop has run N steps. */
  export function stepCountIs(n: number): StopCondition;

  /** Stop when a specific tool is called. */
  export function hasToolCall(toolName: string): StopCondition;

  // ── Message + provider type re-exports ────────────────────────

  /** Shared v5 provider options bag. */
  export type SharedV2ProviderOptions = Record<string, Record<string, unknown>>;

  /** Canonical message type accepted by generate / stream. */
  export type ModelMessage = Message;

  /** Tool choice shape used by AgentCallOptions + generate params. */
  export type ToolChoice =
    | "auto"
    | "none"
    | "required"
    | { type: "tool"; toolName: string };

  /** V2 tool-result part shape (used by tool.toModelOutput hooks). */
  export interface LanguageModelV2ToolResultPart {
    type: "tool-result";
    toolCallId: string;
    toolName: string;
    output: {
      type: "text" | "json" | "error-text" | "error-json" | "content";
      value: unknown;
    };
  }

  /** V2 language model interface — alias of v4 LanguageModel for now. */
  export type LanguageModelV2 = LanguageModel;

  // ── Tool authoring (gap 12) ─────────────────────────────────────

  /**
   * Helper function for inferring the execute args of a tool. Defines a
   * typed tool with an `inputSchema` (Zod or JSON Schema), a
   * `description`, and an `execute` handler. The LLM calls the tool by
   * name during `generateText` / `streamText` when `tools: { name:
   * tool(...) }` is supplied.
   *
   * @example
   *   const weather = tool({
   *     description: "Gets current weather for a city",
   *     inputSchema: z.object({ city: z.string() }),
   *     execute: async ({ city }) => ({ city, tempC: 18 }),
   *   });
   */
  export function tool(config: any): any;

  /**
   * Defines a dynamic tool — one whose input schema is resolved at
   * execute time. Useful when the schema depends on runtime state
   * (capabilities, permissions, user tenant).
   */
  export function dynamicTool(config: any): any;

  /**
   * Create a schema using a JSON Schema. Pass the result to a tool's
   * `inputSchema` or to `generateObject({ schema })`.
   *
   * @param jsonSchema The JSON Schema for the schema.
   * @param options.validate Optional validation function for the schema.
   */
  export function jsonSchema(schema: any, options?: any): any;

  /**
   * Create a schema from a Zod v3 or v4 schema. Same role as
   * {@link jsonSchema} but retains Zod's richer type inference.
   */
  export function zodSchema(schema: any, options?: any): any;

  /**
   * Normalize any FlexibleSchema (Zod / JSON / raw Schema) into the
   * internal `Schema` type. Useful when code receives schemas from
   * multiple sources and has to treat them uniformly.
   */
  export function asSchema(schema: any): any;

  /**
   * Generates a 16-character random string to use for IDs.
   * Not cryptographically secure.
   */
  export function generateId(): string;

  /**
   * Creates an ID generator. The total length of the ID is the sum of
   * the prefix, separator, and random part length. Not cryptographically
   * secure.
   *
   * @param alphabet - The alphabet to use for the ID.
   *   Default `'0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'`.
   * @param prefix - Optional prefix prepended to every generated id.
   * @param separator - Separator between the prefix and the random
   *   part. Default `'-'`.
   * @param size - Size of the random part. Default `16`.
   */
  export function createIdGenerator(options?: { prefix?: string; size?: number; alphabet?: string; separator?: string }): () => string;

  /**
   * Stop condition: true once the agent has executed N steps. Pass to
   * `generateText({ stopWhen: stepCountIs(5) })` to bound a tool loop.
   */
  export function stepCountIs(n: number): (args: any) => boolean;

  /**
   * Stop condition: true as soon as a tool with the given name fires.
   * Use with `stopWhen` to short-circuit once a terminal tool is
   * reached.
   */
  export function hasToolCall(name: string): (args: any) => boolean;

  /**
   * Stop condition that fires at the end of the loop. Internal
   * sentinel — prefer {@link stepCountIs} or {@link hasToolCall} in
   * user code.
   */
  export function isLoopFinished(args: any): boolean;

  // ── Middleware (gap 12) ─────────────────────────────────────────

  /**
   * Wraps a language model with middleware that transforms parameters,
   * wraps generate operations, and wraps stream operations. When
   * multiple middlewares are provided, the first middleware transforms
   * the input first, and the last is wrapped directly around the model.
   *
   * @example
   *   const enhanced = wrapLanguageModel({
   *     model: openai("gpt-4.1"),
   *     middleware: extractReasoningMiddleware({ tagName: "think" }),
   *   });
   *   const { text, reasoning } = await generateText({ model: enhanced, prompt: "..." });
   */
  export function wrapLanguageModel(options: { model: any; middleware: any | any[]; modelId?: string; providerId?: string }): any;

  /**
   * Wraps an embedding model with middleware that transforms parameters
   * and wraps embed operations.
   */
  export function wrapEmbeddingModel(options: { model: any; middleware: any | any[]; modelId?: string; providerId?: string }): any;

  /**
   * Wraps an image model with middleware that transforms parameters
   * and wraps image-generation operations.
   */
  export function wrapImageModel(options: { model: any; middleware: any | any[]; modelId?: string; providerId?: string }): any;

  /**
   * Wraps a provider instance so middleware applies to every language
   * (and optionally image) model resolved through it.
   */
  export function wrapProvider(options: {
    provider: any;
    languageModelMiddleware?: any | any[];
    imageModelMiddleware?: any | any[];
  }): any;

  /**
   * Extracts an XML-tagged reasoning section from the generated text
   * and exposes it as a `reasoning` property on the result. Built for
   * models that emit chain-of-thought in `<think>…</think>` blocks
   * (DeepSeek-R1, friendli, etc).
   *
   * @param tagName The XML tag name to extract reasoning from.
   * @param separator Separator between reasoning and text sections.
   * @param startWithReasoning Whether the model starts with reasoning tokens.
   */
  export function extractReasoningMiddleware(options: { tagName: string; separator?: string; startWithReasoning?: boolean }): any;

  /**
   * Middleware that extracts JSON content from the generated text.
   * Strips markdown code fences by default; customize via the
   * `transform` option.
   */
  export function extractJsonMiddleware(options?: { transform?: (text: string) => string }): any;

  /**
   * Middleware that applies default call settings to every
   * `generateText` / `streamText` call, unless overridden at the call
   * site.
   */
  export function defaultSettingsMiddleware(options: any): any;

  /**
   * Middleware that applies default settings to every `embed` /
   * `embedMany` call.
   */
  export function defaultEmbeddingSettingsMiddleware(options: any): any;

  /**
   * Upgrades a non-streaming model into a streaming one: the generate
   * call runs, the full text is captured, then replayed through the
   * stream interface. Useful when a downstream consumer expects a
   * stream but the provider doesn't support one.
   */
  export function simulateStreamingMiddleware(): any;

  /**
   * Smooths text and reasoning streaming output.
   *
   * @param delayInMs Delay in milliseconds between each chunk.
   *   Defaults to 10ms. Set to `null` to skip.
   * @param chunking Controls how the text is chunked for streaming —
   *   `"word"` (default), `"line"`, a custom RegExp, an Intl.Segmenter
   *   (recommended for CJK languages), or a custom ChunkDetector.
   */
  export function smoothStream(options?: { delayInMs?: number | null; chunking?: any }): any;

  /**
   * Middleware that injects example inputs into the tool descriptions
   * visible to the model. Helps when a tool's schema alone doesn't
   * convey the expected shape.
   */
  export function addToolInputExamplesMiddleware(options?: any): any;

  // ── Provider registry (gap 12) ──────────────────────────────────

  /**
   * Creates a registry for multiple providers with optional middleware.
   * Resolve models via `registry.languageModel("<providerKey>:<modelId>")`.
   *
   * @example
   *   const registry = createProviderRegistry({
   *     openai,
   *     anthropic,
   *   });
   *   const model = registry.languageModel("openai:gpt-4o-mini");
   */
  export function createProviderRegistry(providers: Record<string, any>, options?: any): any;

  /**
   * Creates a custom provider with pre-wired models — useful for
   * project-internal aliasing (`"fast"` → gpt-4o-mini, `"safe"` →
   * claude-sonnet) without exposing provider ids at call sites.
   *
   * @throws {NoSuchModelError} when a requested model is not found and
   *   no `fallbackProvider` is set.
   */
  export function customProvider(options: {
    languageModels?: Record<string, any>;
    embeddingModels?: Record<string, any>;
    imageModels?: Record<string, any>;
    transcriptionModels?: Record<string, any>;
    speechModels?: Record<string, any>;
    fallbackProvider?: any;
  }): any;

  /** @deprecated Use {@link createProviderRegistry}. */
  export function experimental_createProviderRegistry(providers: Record<string, any>, options?: any): any;
  /** @deprecated Use {@link customProvider}. */
  export function experimental_customProvider(options: any): any;

  // ── Message utilities (gap 12) ──────────────────────────────────

  /**
   * Converts an array of UI messages from `useChat` into an array of
   * ModelMessages that can be used with the AI functions (e.g.
   * `streamText`, `generateText`).
   *
   * @param messages The UI messages to convert.
   * @param options.tools The tools to use.
   * @param options.ignoreIncompleteToolCalls Whether to ignore
   *   incomplete tool calls. Default `false`.
   * @param options.convertDataPart Optional function to convert data
   *   parts to text or file model-message parts. Return `undefined` to
   *   drop the part.
   * @returns Promise resolving to the ModelMessage array.
   */
  export function convertToModelMessages(messages: any[], options?: {
    tools?: any;
    ignoreIncompleteToolCalls?: boolean;
    convertDataPart?: (part: any) => any;
  }): Promise<any[]>;

  /**
   * Prunes messages according to a budget (token limit, message count,
   * or a custom predicate) while preserving conversation coherence.
   */
  export function pruneMessages(messages: any[], options?: any): any[];

  /**
   * Validates that an array of UI messages conforms to the expected
   * shape. Throws on the first invalid message.
   */
  export function validateUIMessages(messages: any[], options?: any): Promise<any[]>;

  /**
   * Non-throwing counterpart to {@link validateUIMessages}. Returns a
   * `SafeValidateUIMessagesResult` describing success + values or the
   * failure cause.
   */
  export function safeValidateUIMessages(messages: any[], options?: any): Promise<any>;

  /**
   * Reads a UI-message stream and yields each message delta as it
   * arrives. Typical consumer of the server-side UI message stream.
   */
  export function readUIMessageStream(options: any): AsyncIterable<any>;

  /**
   * Drains a stream to completion, discarding chunks. Use when you
   * need the side effects of streaming (callbacks, token accounting)
   * but don't want to wire up a reader.
   */
  export function consumeStream(stream: any): Promise<void>;

  /**
   * Converts a browser `FileList` into an array of `FileUIPart`s
   * suitable for attaching to a UI message.
   */
  export function convertFileListToFileUIParts(fileList: any, options?: any): Promise<any[]>;

  // ── Media (gap 12) ──────────────────────────────────────────────

  /**
   * Generates images using an image model.
   *
   * @param model The image model to use.
   * @param prompt The prompt used to generate the image.
   * @param n Number of images to generate. Default `1`.
   * @param size Size of the images — `"{width}x{height}"`.
   * @param aspectRatio Aspect ratio — `"{width}:{height}"`.
   * @param seed Seed for reproducible generation.
   * @param maxRetries Maximum retries. `0` disables. Default `2`.
   */
  export function generateImage(options: any): Promise<any>;

  /** @deprecated Use {@link generateImage}. */
  export function experimental_generateImage(options: any): Promise<any>;

  /** Generates a video using a video model. */
  export function experimental_generateVideo(options: any): Promise<any>;

  /** Transcribes audio into text using a transcription model. */
  export function experimental_transcribe(options: any): Promise<any>;

  /**
   * Generates speech audio using a speech model.
   *
   * @param model The speech model to use.
   * @param text The text to convert to speech.
   * @param voice The voice to use.
   * @param outputFormat Output format: `"mp3"`, `"wav"`, etc.
   * @param instructions Delivery instructions (e.g. `"slow and steady"`).
   * @param speed Speech speed multiplier.
   * @param language ISO 639-1 code or `"auto"`.
   */
  export function experimental_generateSpeech(options: any): Promise<any>;

  // ── Misc (gap 12) ───────────────────────────────────────────────

  /**
   * Calculates the cosine similarity between two vectors. Useful for
   * comparing embeddings.
   *
   * @param vector1 The first vector.
   * @param vector2 The second vector.
   * @returns Cosine similarity in `[-1, 1]`, or `0` if either vector
   *   is the zero vector.
   * @throws {InvalidArgumentError} when the vectors differ in length.
   */
  export function cosineSimilarity(a: number[], b: number[]): number;

  /**
   * Creates a ReadableStream that emits the provided values with an
   * optional delay between each value.
   *
   * @param chunks Array of values emitted by the stream.
   * @param initialDelayInMs Initial delay before the first value.
   *   `null` skips the delay entirely; `0` waits 0ms.
   * @param chunkDelayInMs Delay between each chunk. Same `null` vs
   *   `0` semantics as `initialDelayInMs`.
   */
  export function simulateReadableStream<T>(options: {
    chunks: T[];
    initialDelayInMs?: number | null;
    chunkDelayInMs?: number | null;
  }): ReadableStream<T>;

  /**
   * Attempts to parse JSON that may still be mid-stream. Returns a
   * discriminated union on `state`:
   * - `"successful-parse"` — the input was complete JSON.
   * - `"repaired-parse"` — the parser recovered a partial object by
   *   closing unbalanced braces/brackets.
   * - `"failed-parse"` — the input couldn't be recovered.
   * - `"undefined-input"` — the caller passed `undefined` / `null`.
   */
  export function parsePartialJson(raw: string | undefined | null): Promise<{
    value: unknown;
    state: "successful-parse" | "repaired-parse" | "failed-parse" | "undefined-input";
  }>;

  /**
   * Parses an SSE-style JSON event stream into a stream of parsed JSON
   * objects. Validates each event against the optional schema.
   */
  export function parseJsonEventStream(options: any): any;

  // ── Gateway (re-exported from @ai-sdk/gateway) ──────────────────

  /**
   * Default singleton gateway instance. Resolves models by
   * `"<provider>/<modelId>"` against the AI SDK Gateway.
   */
  export const gateway: any;

  /**
   * Builds a gateway provider — typically used to pin the gateway URL
   * or inject custom headers / middleware.
   */
  export function createGateway(options?: any): any;

  // ── Error classes (gap 12) ──────────────────────────────────────
  // Every subclass ships an `isInstance(error)` static guard — use it
  // inside catch blocks to discriminate between failure modes, since
  // cross-realm `instanceof` can be unreliable.

  /**
   * Base class for every AI SDK error. Catch this to treat all AI SDK
   * failures uniformly; narrow via a subclass `isInstance` check to
   * handle specific cases.
   */
  export class AISDKError extends Error {
    static isInstance(error: unknown): error is AISDKError;
    readonly cause?: unknown;
  }

  /**
   * Thrown when an API call to a provider fails — rate limiting, auth,
   * invalid payload, server errors. Carries the HTTP status code and
   * the offending request body for debugging.
   */
  export class APICallError extends AISDKError {
    static isInstance(error: unknown): error is APICallError;
    readonly url?: string;
    readonly requestBodyValues?: unknown;
    readonly statusCode?: number;
    readonly responseBody?: string;
    readonly isRetryable?: boolean;
  }

  /**
   * Thrown by `generateObject` / `streamObject` when the model fails
   * to produce an object that conforms to the schema. Includes the
   * raw text, response, usage, and finish reason for diagnostics.
   */
  export class NoObjectGeneratedError extends AISDKError {
    static isInstance(error: unknown): error is NoObjectGeneratedError;
    readonly text?: string;
    readonly response?: any;
    readonly usage?: any;
    readonly finishReason?: string;
  }

  /** Thrown when a requested model (by id / alias) isn't registered. */
  export class NoSuchModelError extends AISDKError {
    static isInstance(error: unknown): error is NoSuchModelError;
    readonly modelId?: string;
    readonly modelType?: string;
  }

  /** Thrown when the model calls a tool name that isn't in the ToolSet. */
  export class NoSuchToolError extends AISDKError {
    static isInstance(error: unknown): error is NoSuchToolError;
    readonly toolName?: string;
  }

  /** Thrown when a function is called with an argument violating its contract. */
  export class InvalidArgumentError extends AISDKError {
    static isInstance(error: unknown): error is InvalidArgumentError;
  }

  /**
   * Thrown when data content (blob / base64 / URL) isn't in a valid
   * shape for the AI SDK's content-part types.
   */
  export class InvalidDataContentError extends AISDKError {
    static isInstance(error: unknown): error is InvalidDataContentError;
  }

  /** Thrown when a prompt fails structural validation before hitting the provider. */
  export class InvalidPromptError extends AISDKError {
    static isInstance(error: unknown): error is InvalidPromptError;
    readonly prompt?: unknown;
  }

  /**
   * Thrown when the model's tool call has arguments that don't match
   * the tool's `inputSchema`. Includes the raw toolInput for repair
   * strategies.
   */
  export class InvalidToolInputError extends AISDKError {
    static isInstance(error: unknown): error is InvalidToolInputError;
    readonly toolName?: string;
    readonly toolInput?: string;
  }

  /** Thrown when the model returns no content (empty completion). */
  export class NoContentGeneratedError extends AISDKError {
    static isInstance(error: unknown): error is NoContentGeneratedError;
  }

  /** Thrown when `experimental_generateSpeech` receives no audio data. */
  export class NoSpeechGeneratedError extends AISDKError {
    static isInstance(error: unknown): error is NoSpeechGeneratedError;
  }

  /** Thrown when `experimental_transcribe` produces no transcript text. */
  export class NoTranscriptGeneratedError extends AISDKError {
    static isInstance(error: unknown): error is NoTranscriptGeneratedError;
  }

  /** Thrown when `experimental_generateVideo` produces no video data. */
  export class NoVideoGeneratedError extends AISDKError {
    static isInstance(error: unknown): error is NoVideoGeneratedError;
  }

  /**
   * Thrown after the retry budget is exhausted. `.errors` holds the
   * chain of underlying failures; `.reason` distinguishes timeout,
   * max-retries, or non-retryable.
   */
  export class RetryError extends AISDKError {
    static isInstance(error: unknown): error is RetryError;
    readonly reason?: string;
    readonly errors?: unknown[];
  }

  /**
   * Thrown from a `toolCallRepair` hook when repair itself fails.
   */
  export class ToolCallRepairError extends AISDKError {
    static isInstance(error: unknown): error is ToolCallRepairError;
  }

  /** Thrown when a value fails schema-based type validation. */
  export class TypeValidationError extends AISDKError {
    static isInstance(error: unknown): error is TypeValidationError;
    readonly value?: unknown;
  }

  /** Thrown when message shape conversion (UI ↔ Model / v4 ↔ v5) fails. */
  export class MessageConversionError extends AISDKError {
    static isInstance(error: unknown): error is MessageConversionError;
  }

  /**
   * Thrown when a message references tool results that never arrived.
   * Often indicates a dropped tool call / response pair.
   */
  export class MissingToolResultsError extends AISDKError {
    static isInstance(error: unknown): error is MissingToolResultsError;
  }

  /**
   * Thrown when the provider can't resolve an API key from env / config.
   */
  export class LoadAPIKeyError extends AISDKError {
    static isInstance(error: unknown): error is LoadAPIKeyError;
  }

  /** Thrown when a tool-approval response is malformed. */
  export class InvalidToolApprovalError extends AISDKError {
    static isInstance(error: unknown): error is InvalidToolApprovalError;
  }

  /**
   * Thrown when a tool-approval response references a toolCallId that
   * no longer exists on the run — common after a suspend/resume cycle
   * with mismatched ids.
   */
  export class ToolCallNotFoundForApprovalError extends AISDKError {
    static isInstance(error: unknown): error is ToolCallNotFoundForApprovalError;
  }
}
