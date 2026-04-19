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
    /** Maximum number of tokens to generate. */
    maxOutputTokens?: number;
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

  /** Tool definition for AI SDK. */
  export interface ToolDefinition {
    description?: string;
    parameters: ZodType;
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
    /** Callback per stream chunk. */
    onChunk?: (event: { chunk: StreamPart }) => void;
    /** Error callback. */
    onError?: (event: { error: unknown }) => void;
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

  export interface GenerateObjectResult<T = unknown> {
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

  export function generateObject<T = unknown>(params: GenerateObjectParams): Promise<GenerateObjectResult<T>>;

  // ── streamObject ──────────────────────────────────────────────

  export interface StreamObjectParams extends GenerateObjectParams {
    /** Error callback. */
    onError?: (event: { error: unknown }) => void;
    /** Finish callback. */
    onFinish?: (event: { object: unknown; usage: Usage }) => void;
  }

  export interface StreamObjectResult<T = unknown> {
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

  export function streamObject<T = unknown>(params: StreamObjectParams): StreamObjectResult<T>;

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
}
