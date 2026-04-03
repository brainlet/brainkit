package cmd

// Type definitions for brainkit modules — written by `brainkit new module`.
// Derived from the Compartment endowments in runtime/kit_runtime.js
// and the module exports in runtime/*_module.js.

const kitDTS = `// brainkit "kit" module — infrastructure APIs

declare module "kit" {
  interface BusMessage {
    payload: any;
    replyTo: string;
    correlationId: string;
    topic: string;
    callerId: string;
    reply(data: any): void;
    send(data: any): void;
    stream: {
      text(chunk: string): void;
      progress(value: number, message?: string): void;
      object(partial: any): void;
      event(name: string, data?: any): void;
      error(message: string): void;
      end(finalData?: any): void;
    };
  }

  interface PublishResult {
    replyTo: string;
    correlationId: string;
  }

  interface ResourceEntry {
    type: string;
    id: string;
    name: string;
    source: string;
    createdAt: number;
  }

  interface ToolInfo {
    name: string;
    shortName: string;
    description: string;
    inputSchema?: any;
  }

  export const bus: {
    publish(topic: string, data?: any): PublishResult;
    emit(topic: string, data?: any): void;
    subscribe(topic: string, handler: (msg: BusMessage) => void | Promise<void>): string;
    on(localTopic: string, handler: (msg: BusMessage) => void | Promise<void>): string;
    unsubscribe(subscriptionId: string): void;
    sendTo(service: string, localTopic: string, data?: any): PublishResult;
    schedule(expression: string, topic: string, data?: any): string;
    unschedule(scheduleId: string): void;
  };

  export const kit: {
    register(type: "tool" | "agent" | "workflow" | "memory", name: string, ref: any): void;
    unregister(type: string, name: string): void;
    list(type?: string): ResourceEntry[];
    readonly source: string;
    readonly namespace: string;
    readonly callerId: string;
  };

  export function model(provider: string, modelId: string): any;
  export function embeddingModel(provider: string, modelId: string): any;
  export function provider(name: string): any;
  export function storage(name: string): any;
  export function vectorStore(name: string): any;

  export const registry: {
    has(category: "provider" | "vectorStore" | "storage", name: string): boolean;
    list(category: "provider" | "vectorStore" | "storage"): any[];
    resolve(category: string, name: string): any;
    register(category: string, name: string, config: any): void;
    unregister(category: string, name: string): void;
  };

  export const tools: {
    call(name: string, input: any): Promise<any>;
    list(namespace?: string): ToolInfo[];
    resolve(name: string): ToolInfo | null;
  };

  export const secrets: {
    get(name: string): string;
  };

  export function output(value: any): void;

  export const mcp: {
    listTools(server?: string): Array<{ name: string; server: string; description: string }>;
    callTool(server: string, tool: string, args?: any): Promise<any>;
  };

  export const fs: typeof import("fs");

  export function generateWithApproval(
    agent: any,
    promptOrMessages: string | any[],
    options: { approvalTopic: string; timeout?: number; [key: string]: any },
  ): Promise<any>;
}
`

const aiDTS = `// brainkit "ai" module — AI SDK

declare module "ai" {
  interface Message {
    role: "system" | "user" | "assistant" | "tool";
    content: string | any[];
  }

  interface Usage {
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
  }

  interface ResponseMeta {
    id: string;
    modelId: string;
    timestamp?: string;
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
    finishReason: string;
    usage: Usage;
    toolCalls?: ToolCall[];
    toolResults?: ToolResult[];
    stepType: string;
    isContinued: boolean;
  }

  interface GenerateTextResult {
    text: string;
    reasoning?: string;
    finishReason: string;
    usage: Usage;
    response: ResponseMeta;
    toolCalls?: ToolCall[];
    toolResults?: ToolResult[];
    steps?: StepResult[];
  }

  interface StreamTextResult {
    textStream: AsyncIterable<string>;
    fullStream: AsyncIterable<any>;
    text: Promise<string>;
    usage: Promise<Usage>;
    finishReason: Promise<string>;
    response: Promise<ResponseMeta>;
  }

  interface GenerateObjectResult<T = any> {
    object: T;
    finishReason: string;
    usage: Usage;
    response: ResponseMeta;
  }

  interface StreamObjectResult<T = any> {
    partialObjectStream: AsyncIterable<Partial<T>>;
    object: Promise<T>;
    usage: Promise<Usage>;
    response: Promise<ResponseMeta>;
  }

  interface EmbedResult {
    embedding: number[];
    usage: { tokens: number };
  }

  interface EmbedManyResult {
    embeddings: number[][];
    usage: { tokens: number };
  }

  interface GenerateTextOptions {
    model: any;
    prompt?: string;
    system?: string;
    messages?: Message[];
    tools?: Record<string, any>;
    toolChoice?: "auto" | "none" | "required" | { type: "tool"; toolName: string };
    maxSteps?: number;
    maxTokens?: number;
    temperature?: number;
    topP?: number;
    stopSequences?: string[];
    providerOptions?: Record<string, any>;
  }

  interface GenerateObjectOptions {
    model: any;
    prompt?: string;
    system?: string;
    messages?: Message[];
    schema: any;
    schemaName?: string;
    schemaDescription?: string;
    mode?: "auto" | "json" | "tool";
    maxTokens?: number;
    temperature?: number;
    providerOptions?: Record<string, any>;
  }

  export function generateText(options: GenerateTextOptions): Promise<GenerateTextResult>;
  export function streamText(options: GenerateTextOptions): StreamTextResult;
  export function generateObject<T = any>(options: GenerateObjectOptions): Promise<GenerateObjectResult<T>>;
  export function streamObject<T = any>(options: GenerateObjectOptions): StreamObjectResult<T>;
  export function embed(options: { model: any; value: string }): Promise<EmbedResult>;
  export function embedMany(options: { model: any; values: string[] }): Promise<EmbedManyResult>;

  export const z: {
    object(shape: Record<string, any>): any;
    string(): any;
    number(): any;
    boolean(): any;
    array(item: any): any;
    enum(values: [string, ...string[]]): any;
    optional(schema: any): any;
    any(): any;
    [key: string]: any;
  };
}
`

const agentDTS = `// brainkit "agent" module — Mastra framework

declare module "agent" {
  // Core
  export const Agent: any;
  export function createTool(config: {
    id: string;
    description: string;
    inputSchema: any;
    execute: (input: any, context?: any) => Promise<any>;
  }): any;
  export function createWorkflow(config: {
    id: string;
    inputSchema?: any;
    outputSchema?: any;
  }): any;
  export function createStep(config: {
    id: string;
    inputSchema?: any;
    outputSchema?: any;
    execute: (context: any) => Promise<any>;
  }): any;
  export const Memory: any;
  export const RequestContext: any;
  export const z: any;

  // Storage backends
  export const InMemoryStore: any;
  export const LibSQLStore: any;
  export const UpstashStore: any;
  export const PostgresStore: any;
  export const MongoDBStore: any;

  // Vector backends
  export const LibSQLVector: any;
  export const PgVector: any;
  export const MongoDBVector: any;
  export const ModelRouterEmbeddingModel: any;

  // Workspace
  export const Workspace: any;
  export const LocalFilesystem: any;
  export const LocalSandbox: any;

  // RAG
  export const MDocument: any;
  export const GraphRAG: any;
  export function createVectorQueryTool(config: any): any;
  export function createDocumentChunkerTool(config: any): any;
  export function createGraphRAGTool(config: any): any;
  export function rerank(options: any): Promise<any>;
  export function rerankWithScorer(options: any): Promise<any>;

  // Observability
  export const Observability: any;
  export const DefaultExporter: any;
  export const SensitiveDataFilter: any;

  // Evals
  export function createScorer(config: any): any;
  export function runEvals(config: any): Promise<any>;
}
`

const testDTS = `// brainkit "test" module — test framework

declare module "test" {
  interface Expectation {
    toBe(expected: any): void;
    toEqual(expected: any): void;
    toContain(sub: string): void;
    toMatch(pattern: string | RegExp): void;
    toBeTruthy(): void;
    toBeFalsy(): void;
    toBeDefined(): void;
    toBeNull(): void;
    toBeGreaterThan(n: number): void;
    toBeLessThan(n: number): void;
    toHaveLength(n: number): void;
    toHaveProperty(key: string): void;
    toThrow(message?: string): void;
    not: Expectation;
  }

  export function test(name: string, fn: () => void | Promise<void>): void;
  export function describe(name: string, fn: () => void): void;
  export function expect(value: any): Expectation;
  export function beforeAll(fn: () => void | Promise<void>): void;
  export function afterAll(fn: () => void | Promise<void>): void;
  export function beforeEach(fn: () => void | Promise<void>): void;
  export function afterEach(fn: () => void | Promise<void>): void;
  export function deploy(source: string, code: string): Promise<any>;
  export function deployFile(path: string): Promise<any>;
  export function sleep(ms: number): Promise<void>;
  export function timeout(ms: number): void;
  export function sendTo(service: string, topic: string, data?: any, timeoutMs?: number): Promise<any>;
  export function evaluate(service: string, topic: string, cases: Array<{
    input: any;
    expect?: Record<string, any>;
    timeout?: number;
  }>): Promise<{
    total: number;
    passed: number;
    failed: number;
    accuracy: number;
    totalDuration: number;
    items: any[];
  }>;
  export function runWorkflow(workflowId: string, opts?: {
    input?: any;
    hostResults?: any;
  }): Promise<{
    status: string;
    output?: any;
    steps: any[];
    runId: string;
  }>;
}
`
