# AI SDK — API Reference for brainkit

> `import { generateText, streamText, generateObject, streamObject, embed, embedMany, z } from "ai";`
> Types from `kit/runtime/ai.d.ts`, verified against real AI SDK.

## generateText

```typescript
function generateText(params: {
    model: LanguageModel;
    prompt?: string;
    messages?: Message[];
    system?: string;
    tools?: Record<string, ToolDefinition>;
    maxSteps?: number;
    maxTokens?: number;
    temperature?: number;
    topP?: number;
    topK?: number;
    stopSequences?: string[];
    seed?: number;
    abortSignal?: AbortSignal;
    headers?: Record<string, string>;
    providerOptions?: Record<string, Record<string, unknown>>;
    onStepFinish?: (step: StepResult) => void | Promise<void>;
}): Promise<GenerateTextResult>;

interface GenerateTextResult {
    text: string;
    reasoning?: string;
    sources?: Source[];
    toolCalls: ToolCall[];
    toolResults: ToolResult[];
    finishReason: FinishReason;
    usage: Usage;
    steps: StepResult[];
    response: ResponseMeta;
    warnings?: Warning[];
    files?: GeneratedFile[];
    providerMetadata?: ProviderMetadata;
}
```

## streamText

```typescript
function streamText(params: {
    model: LanguageModel;
    prompt?: string;
    messages?: Message[];
    system?: string;
    tools?: Record<string, ToolDefinition>;
    maxSteps?: number;
    maxTokens?: number;
    temperature?: number;
    onChunk?: (chunk: { type: string; [key: string]: unknown }) => void;
    onFinish?: (result: StreamTextResult) => void;
    abortSignal?: AbortSignal;
}): Promise<StreamTextHandle>;

interface StreamTextHandle {
    textStream: AsyncIterable<string>;
    fullStream: AsyncIterable<StreamPart>;
    text: Promise<string>;
    usage: Promise<Usage>;
    finishReason: Promise<FinishReason>;
    response: Promise<ResponseMeta>;
    steps: Promise<StepResult[]>;
}
```

## generateObject

```typescript
function generateObject<T>(params: {
    model: LanguageModel;
    prompt?: string;
    messages?: Message[];
    system?: string;
    schema: ZodType<T>;
    schemaName?: string;
    schemaDescription?: string;
    mode?: "auto" | "json" | "tool";
    maxTokens?: number;
    temperature?: number;
}): Promise<GenerateObjectResult<T>>;

interface GenerateObjectResult<T> {
    object: T;
    finishReason: FinishReason;
    usage: Usage;
    response: ResponseMeta;
    warnings?: Warning[];
}
```

## streamObject

```typescript
function streamObject<T>(params: {
    model: LanguageModel;
    prompt?: string;
    messages?: Message[];
    system?: string;
    schema: ZodType<T>;
    mode?: "auto" | "json" | "tool";
    onFinish?: (result: { object: T; usage: Usage }) => void;
}): Promise<StreamObjectHandle<T>>;

interface StreamObjectHandle<T> {
    partialObjectStream: AsyncIterable<Partial<T>>;
    object: Promise<T>;
    usage: Promise<Usage>;
    finishReason: Promise<FinishReason>;
}
```

## embed

```typescript
function embed(params: {
    model: EmbeddingModel;
    value: string;
}): Promise<EmbedResult>;

interface EmbedResult {
    embedding: number[];
    usage: { tokens: number };
}
```

## embedMany

```typescript
function embedMany(params: {
    model: EmbeddingModel;
    values: string[];
}): Promise<EmbedManyResult>;

interface EmbedManyResult {
    embeddings: number[][];
    usage: { tokens: number };
}
```

## Shared Types

```typescript
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
}

type ContentPart =
    | { type: "text"; text: string }
    | { type: "image"; image: string | Uint8Array; mimeType?: string }
    | { type: "tool-call"; toolCallId: string; toolName: string; args: Record<string, unknown> }
    | { type: "tool-result"; toolCallId: string; toolName: string; result: unknown };

interface Warning { type: string; message: string; }
interface GeneratedFile { data: Uint8Array; mimeType: string; }

type ProviderMetadata = Record<string, Record<string, unknown>>;
type ProviderOptions = Record<string, Record<string, unknown>>;
```

## z (Zod v4)

```typescript
const z: {
    string(): ZodString;
    number(): ZodNumber;
    boolean(): ZodBoolean;
    object(shape: Record<string, ZodType>): ZodObject;
    array(element: ZodType): ZodArray;
    enum(values: [string, ...string[]]): ZodEnum;
    any(): ZodAny;
    optional(): ZodOptional;
    nullable(): ZodNullable;
    union(types: ZodType[]): ZodUnion;
    literal(value: string | number | boolean): ZodLiteral;
    // ... full Zod v4 API
};
```

Same `z` instance is available from both `"ai"` and `"agent"` modules.

## Tool Definition (for AI SDK functions)

```typescript
interface ToolDefinition {
    description: string;
    parameters: ZodType;
    execute: (args: any) => Promise<any>;
}
```

Used in `generateText({ tools: { myTool: { description, parameters, execute } } })`. Different from Mastra's `createTool` — AI SDK tools are plain objects, Mastra tools are class instances.
