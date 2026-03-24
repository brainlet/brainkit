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
    default(value: any): ZodType;
    describe(description: string): ZodType;
    array(): ZodType;
    or(other: ZodType): ZodType;
    and(other: ZodType): ZodType;
    transform(fn: (val: any) => any): ZodType;
    refine(fn: (val: any) => boolean, message?: string): ZodType;
    parse(value: any): any;
    safeParse(value: any): { success: boolean; data?: any; error?: any };
  }

  // ── CallSettings (shared across all functions) ────────────────

  interface CallSettings {
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

  interface Usage {
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
  }

  interface ResponseMeta {
    id: string;
    modelId: string;
    timestamp: Date;
    headers?: Record<string, string>;
  }

  interface ToolCall {
    toolCallId: string;
    toolName: string;
    args: any;
  }

  interface ToolResult {
    toolCallId: string;
    toolName: string;
    args: any;
    result: any;
  }

  interface StepResult {
    text: string;
    reasoning?: string;
    toolCalls: ToolCall[];
    toolResults: ToolResult[];
    finishReason: FinishReason;
    usage: Usage;
    stepType: string;
    isContinued: boolean;
  }

  interface Source {
    id: string;
    url?: string;
    title?: string;
    sourceType?: string;
    providerMetadata?: Record<string, any>;
  }

  /** Language model instance (returned by provider factory). */
  type LanguageModel = any;

  /** Embedding model instance. */
  type EmbeddingModel = any;

  /** Tool definition for AI SDK. */
  interface ToolDefinition {
    description?: string;
    parameters: ZodType;
    execute?: (args: any, options?: any) => Promise<any>;
  }

  /** Provider options keyed by provider name. */
  type ProviderOptions = Record<string, Record<string, any>>;

  // ── generateText ──────────────────────────────────────────────

  interface GenerateTextParams extends CallSettings {
    /** The language model to use. */
    model: LanguageModel;
    /** Simple text prompt. */
    prompt?: string;
    /** System message. */
    system?: string;
    /** Conversation messages. */
    messages?: Array<{ role: string; content: any }>;
    /** Tool definitions. */
    tools?: Record<string, ToolDefinition>;
    /** Tool selection strategy. */
    toolChoice?: "auto" | "none" | "required" | { type: "tool"; toolName: string };
    /** Limit which tools are active per step. */
    activeTools?: string[];
    /** Max steps for multi-step generation. */
    maxSteps?: number;
    /** Provider-specific options. */
    providerOptions?: ProviderOptions;
    /** Structured output specification. */
    output?: any;
    /** Callback: each step finishes. */
    onStepFinish?: (event: StepResult) => void | Promise<void>;
    /** Callback: all steps done. */
    onFinish?: (event: GenerateTextResult) => void | Promise<void>;
  }

  interface GenerateTextResult {
    /** Generated text from the last step. */
    text: string;
    /** Reasoning text (if extractReasoningMiddleware used). */
    reasoningText?: string;
    /** Tool calls made in the last step. */
    toolCalls: ToolCall[];
    /** Tool results from the last step. */
    toolResults: ToolResult[];
    /** Why generation stopped. */
    finishReason: FinishReason;
    /** Token usage. */
    usage: Usage;
    /** All steps in multi-step generation. */
    steps: StepResult[];
    /** Response metadata. */
    response: ResponseMeta;
    /** Generated files (images, etc). */
    files: any[];
    /** Source attributions. */
    sources: Source[];
    /** Warnings from the provider. */
    warnings: any[];
    /** Provider-specific metadata. */
    providerMetadata?: Record<string, any>;
  }

  export function generateText(params: GenerateTextParams): Promise<GenerateTextResult>;

  // ── streamText ────────────────────────────────────────────────

  interface StreamTextParams extends GenerateTextParams {
    /** Callback per stream chunk. */
    onChunk?: (event: { chunk: any }) => void;
    /** Error callback. */
    onError?: (event: { error: unknown }) => void;
  }

  interface StreamTextResult {
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
    | { type: "tool-call"; toolCallId: string; toolName: string; args: any }
    | { type: "tool-result"; toolCallId: string; toolName: string; result: any }
    | { type: "step-finish"; finishReason: FinishReason; usage: Usage }
    | { type: "finish"; finishReason: FinishReason; usage: Usage }
    | { type: "error"; error: unknown };

  export function streamText(params: StreamTextParams): StreamTextResult;

  // ── generateObject ────────────────────────────────────────────

  interface GenerateObjectParams extends CallSettings {
    /** The language model to use. */
    model: LanguageModel;
    /** Simple text prompt. */
    prompt?: string;
    /** System message. */
    system?: string;
    /** Conversation messages. */
    messages?: Array<{ role: string; content: any }>;
    /** Output schema (Zod or JSON Schema). */
    schema?: ZodType | any;
    /** Optional name for the schema. */
    schemaName?: string;
    /** Optional description for the schema. */
    schemaDescription?: string;
    /** Generation strategy: "auto" | "json" | "tool". */
    mode?: "auto" | "json" | "tool";
    /** Output type. */
    output?: "object" | "array" | "enum" | "no-schema";
    /** Enum values (for output: "enum"). */
    enum?: any[];
    /** Provider-specific options. */
    providerOptions?: ProviderOptions;
  }

  interface GenerateObjectResult<T = any> {
    /** The generated object. */
    object: T;
    /** Why generation stopped. */
    finishReason: FinishReason;
    /** Token usage. */
    usage: Usage;
    /** Response metadata. */
    response: ResponseMeta;
    /** Warnings. */
    warnings: any[];
    /** Provider-specific metadata. */
    providerMetadata?: Record<string, any>;
  }

  export function generateObject<T = any>(params: GenerateObjectParams): Promise<GenerateObjectResult<T>>;

  // ── streamObject ──────────────────────────────────────────────

  interface StreamObjectParams extends GenerateObjectParams {
    /** Error callback. */
    onError?: (event: { error: unknown }) => void;
    /** Finish callback. */
    onFinish?: (event: { object: any; usage: Usage }) => void;
  }

  interface StreamObjectResult<T = any> {
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

  interface EmbedParams {
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

  interface EmbedResult {
    /** The embedding vector. */
    embedding: number[];
    /** Token usage. */
    usage: { tokens: number };
  }

  export function embed(params: EmbedParams): Promise<EmbedResult>;

  // ── embedMany ─────────────────────────────────────────────────

  interface EmbedManyParams {
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

  interface EmbedManyResult {
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

  /** Language model middleware type. */
  type LanguageModelMiddleware = any;

  // ── Tool utilities ────────────────────────────────────────────

  /** Define a tool with schema + execute function. */
  export function tool<T>(definition: {
    description?: string;
    parameters: ZodType;
    execute?: (args: T, options?: any) => Promise<any>;
  }): ToolDefinition;

  /** Convert a JSON Schema object to an AI SDK schema. */
  export function jsonSchema(schema: any): any;
}
