/**
 * Global types available inside brainkit SES Compartments.
 * These are injected as endowments — not imported via ES modules.
 */

// ── Node.js globals ────────────────────────────────────────────

declare var Buffer: {
  from(data: string | ArrayBuffer | Uint8Array | number[] | ArrayBufferView, encoding?: string, length?: number): any;
  alloc(size: number, fill?: number): any;
  allocUnsafe(size: number): any;
  isBuffer(obj: any): boolean;
  isEncoding(enc: string): boolean;
  byteLength(str: string | Uint8Array | ArrayBuffer, encoding?: string): number;
  concat(bufs: any[], totalLength?: number): any;
  compare(a: any, b: any): number;
  poolSize: number;
};

declare var process: {
  env: Record<string, string | undefined>;
  cwd(): string;
  version: string;
  versions: Record<string, string>;
  platform: string;
  arch: string;
  pid: number;
  argv: string[];
  execPath: string;
  title: string;
  nextTick(fn: (...args: any[]) => void, ...args: any[]): void;
  hrtime(prev?: [number, number]): [number, number];
  stdout: { write(s: string): boolean };
  stderr: { write(s: string): boolean };
  on(event: string, listener: (...args: any[]) => void): typeof process;
  once(event: string, listener: (...args: any[]) => void): typeof process;
  off(event: string, listener: (...args: any[]) => void): typeof process;
  emit(event: string, ...args: any[]): boolean;
  emitWarning(msg: string): void;
  getuid(): number;
  getgid(): number;
  geteuid(): number;
  getegid(): number;
  exit(code?: number): void;
  umask(mask?: number): number;
  uptime(): number;
  memoryUsage(): { rss: number; heapTotal: number; heapUsed: number; external: number };
  cpuUsage(): { user: number; system: number };
};

declare class EventEmitter {
  constructor();
  on(event: string, listener: (...args: any[]) => void): this;
  addListener(event: string, listener: (...args: any[]) => void): this;
  prependListener(event: string, listener: (...args: any[]) => void): this;
  once(event: string, listener: (...args: any[]) => void): this;
  prependOnceListener(event: string, listener: (...args: any[]) => void): this;
  emit(event: string, ...args: any[]): boolean;
  removeListener(event: string, listener: (...args: any[]) => void): this;
  off(event: string, listener: (...args: any[]) => void): this;
  removeAllListeners(event?: string): this;
  setMaxListeners(n: number): this;
  getMaxListeners(): number;
  listenerCount(event: string): number;
  listeners(event: string): Function[];
  rawListeners(event: string): Function[];
  eventNames(): string[];
  static captureRejections: boolean;
  static defaultMaxListeners: number;
  static setMaxListeners(...args: any[]): void;
  static listenerCount(emitter: EventEmitter, event: string): number;
}

// ── Node.js module globals (available in SES Compartments) ─────
// These match real Node.js module names. Set directly by jsbridge polyfills.

declare var stream: {
  Readable: any;
  Writable: any;
  Duplex: any;
  Transform: any;
  PassThrough: any;
  pipeline: (...args: any[]) => void;
  finished: (stream: any, cb?: (err?: Error) => void) => void;
  Stream: any;
};

declare var crypto: {
  createHash(alg: string): { update(data: any, enc?: string): any; digest(enc?: string): any; copy(): any };
  createHmac(alg: string, key: any): { update(data: any, enc?: string): any; digest(enc?: string): any };
  pbkdf2Sync(password: any, salt: any, iterations: number, keylen: number, hash: string): any;
  pbkdf2(password: any, salt: any, iterations: number, keylen: number, hash: string, cb: (err: any, key: any) => void): void;
  randomBytes(n: number, cb?: (err: any, buf: any) => void): any;
  randomFillSync(buf: any): any;
  randomInt(min: number, max?: number): number;
  timingSafeEqual(a: any, b: any): boolean;
  getHashes(): string[];
  getCiphers(): string[];
  getFips(): number;
  webcrypto: typeof crypto;
};

declare var net: {
  Socket: any;
  createConnection: (...args: any[]) => any;
  connect: (...args: any[]) => any;
  createServer: (...args: any[]) => any;
  Server: any;
  isIP(input: string): number;
  isIPv4(input: string): boolean;
  isIPv6(input: string): boolean;
};

declare var os: {
  platform(): string;
  arch(): string;
  tmpdir(): string;
  homedir(): string;
  hostname(): string;
  type(): string;
  cpus(): Array<{ model: string; speed: number }>;
  EOL: string;
  endianness(): string;
  release(): string;
  totalmem(): number;
  freemem(): number;
  uptime(): number;
  loadavg(): [number, number, number];
  networkInterfaces(): Record<string, any>;
  userInfo(): { username: string; uid: number; gid: number; shell: string; homedir: string };
};

declare var dns: {
  lookup(hostname: string, cb: (err: any, addr: string, family: number) => void): void;
  lookup(hostname: string, options: any, cb: (err: any, addr: string, family: number) => void): void;
  resolve4(hostname: string, cb: (err: any, addrs: string[]) => void): void;
  Resolver: any;
  promises: {
    lookup(hostname: string): Promise<{ address: string; family: number }>;
    resolve4(hostname: string): Promise<string[]>;
    resolveSrv(hostname: string): Promise<any[]>;
    resolveCname(hostname: string): Promise<string[]>;
    resolvePtr(hostname: string): Promise<string[]>;
  };
};

declare var zlib: {
  inflate(buf: any, cb: (err: any, result: any) => void): void;
  deflate(buf: any, cb: (err: any, result: any) => void): void;
  deflate(buf: any, opts: any, cb: (err: any, result: any) => void): void;
  gunzip(buf: any, cb: (err: any, result: any) => void): void;
  gzip(buf: any, cb: (err: any, result: any) => void): void;
  inflateSync(buf: any): any;
  deflateSync(buf: any, opts?: any): any;
  gunzipSync(buf: any): any;
  gzipSync(buf: any): any;
  inflateRaw(buf: any, cb: (err: any, result: any) => void): void;
  deflateRaw(buf: any, cb: (err: any, result: any) => void): void;
  inflateRawSync(buf: any): any;
  deflateRawSync(buf: any, opts?: any): any;
  createGzip(): any;
  createGunzip(): any;
  createDeflate(opts?: any): any;
  createInflate(): any;
  constants: {
    Z_NO_COMPRESSION: number;
    Z_BEST_SPEED: number;
    Z_BEST_COMPRESSION: number;
    Z_DEFAULT_COMPRESSION: number;
    Z_DEFAULT_STRATEGY: number;
  };
};

declare var child_process: {
  exec(command: string): Promise<{ stdout: string; stderr: string; exitCode: number }>;
  execSync(command: string): any;
  execFileSync(file: string, args?: string[], options?: { cwd?: string }): any;
  spawnSync(command: string, args?: string[], options?: { cwd?: string }): { stdout: string; stderr: string; status: number; error: any };
  spawn(command: string, args?: string[], cwd?: string): {
    pid: number;
    readLine(): Promise<string | null>;
    readChunk(): Promise<string | null>;
    write(data: string): Promise<boolean>;
    wait(): Promise<number>;
    kill(): void;
  };
};

// ── Core web globals not covered by es2022 lib ─────────────────
// When a .ts deployment's tsconfig doesn't include `dom`, these
// are still needed — brainkit polyfills ship them all, so we
// declare enough of each for IDE completion. Shapes deliberately
// loose — real usage leans on DOM types when `lib: ["dom"]`.

/** AbortController / AbortSignal — provided by jsbridge/abort.go. */
declare class AbortController {
  constructor();
  readonly signal: AbortSignal;
  abort(reason?: unknown): void;
}
declare class AbortSignal {
  readonly aborted: boolean;
  readonly reason: unknown;
  throwIfAborted(): void;
  addEventListener(event: "abort", listener: (ev: any) => void): void;
  removeEventListener(event: "abort", listener: (ev: any) => void): void;
  static abort(reason?: unknown): AbortSignal;
  static timeout(ms: number): AbortSignal;
  onabort?: ((ev: any) => void) | null;
}

/** Web Streams — provided by jsbridge/streams.go. */
declare class ReadableStream<T = any> {
  constructor(source?: any, strategy?: any);
  readonly locked: boolean;
  cancel(reason?: any): Promise<void>;
  getReader(): any;
  pipeThrough(transform: any, options?: any): ReadableStream<T>;
  pipeTo(destination: any, options?: any): Promise<void>;
  tee(): [ReadableStream<T>, ReadableStream<T>];
  [Symbol.asyncIterator](): AsyncIterableIterator<T>;
}
declare class WritableStream<T = any> {
  constructor(sink?: any, strategy?: any);
  readonly locked: boolean;
  abort(reason?: any): Promise<void>;
  close(): Promise<void>;
  getWriter(): any;
}
declare class TransformStream<I = any, O = any> {
  constructor(transformer?: any, writableStrategy?: any, readableStrategy?: any);
  readonly readable: ReadableStream<O>;
  readonly writable: WritableStream<I>;
}

/** fetch Response + Request — provided by jsbridge/fetch.go. */
declare class Response {
  constructor(body?: any, init?: { status?: number; statusText?: string; headers?: any });
  readonly status: number;
  readonly statusText: string;
  readonly ok: boolean;
  readonly headers: any;
  readonly url: string;
  readonly body: ReadableStream | null;
  readonly bodyUsed: boolean;
  text(): Promise<string>;
  json(): Promise<any>;
  arrayBuffer(): Promise<ArrayBuffer>;
  clone(): Response;
  static json(data: any, init?: any): Response;
}
declare class Request {
  constructor(input: string | Request, init?: any);
  readonly url: string;
  readonly method: string;
  readonly headers: any;
  text(): Promise<string>;
  json(): Promise<any>;
  arrayBuffer(): Promise<ArrayBuffer>;
}
declare class Headers {
  constructor(init?: Record<string, string> | [string, string][] | Headers);
  get(name: string): string | null;
  set(name: string, value: string): void;
  append(name: string, value: string): void;
  delete(name: string): void;
  has(name: string): boolean;
  forEach(cb: (value: string, key: string, parent: Headers) => void): void;
  entries(): IterableIterator<[string, string]>;
  keys(): IterableIterator<string>;
  values(): IterableIterator<string>;
  [Symbol.iterator](): IterableIterator<[string, string]>;
}

/** fetch — provided by jsbridge/fetch.go. */
declare const fetch: (input: string | Request, init?: any) => Promise<Response>;

/** URL + URLSearchParams — provided by jsbridge/url.go. */
declare class URL {
  constructor(url: string, base?: string | URL);
  href: string;
  origin: string;
  protocol: string;
  username: string;
  password: string;
  host: string;
  hostname: string;
  port: string;
  pathname: string;
  search: string;
  searchParams: URLSearchParams;
  hash: string;
  toString(): string;
  toJSON(): string;
  static createObjectURL(blob: Blob): string;
  static revokeObjectURL(id: string): void;
}
declare class URLSearchParams {
  constructor(init?: string | Record<string, string> | [string, string][] | URLSearchParams);
  get(name: string): string | null;
  getAll(name: string): string[];
  set(name: string, value: string): void;
  append(name: string, value: string): void;
  delete(name: string): void;
  has(name: string): boolean;
  toString(): string;
  forEach(cb: (value: string, key: string, parent: URLSearchParams) => void): void;
  entries(): IterableIterator<[string, string]>;
  keys(): IterableIterator<string>;
  values(): IterableIterator<string>;
  [Symbol.iterator](): IterableIterator<[string, string]>;
}

/** TextEncoder / TextDecoder — provided by jsbridge/encoding.go. */
declare class TextEncoder {
  readonly encoding: "utf-8";
  encode(s?: string): Uint8Array;
  encodeInto(s: string, dest: Uint8Array): { read: number; written: number };
}
declare class TextDecoder {
  constructor(label?: string, options?: { fatal?: boolean; ignoreBOM?: boolean });
  readonly encoding: string;
  readonly fatal: boolean;
  readonly ignoreBOM: boolean;
  decode(input?: ArrayBuffer | ArrayBufferView, options?: { stream?: boolean }): string;
}

/** Base64 helpers — provided by jsbridge/encoding.go. */
declare const btoa: (s: string) => string;
declare const atob: (s: string) => string;

/** structuredClone — provided by jsbridge/structured_clone.go. */
declare const structuredClone: <T>(value: T) => T;

// ── Web-standard audio + media polyfills ───────────────────────
// All live on globalThis via internal/jsbridge/audio.go,
// websocket.go, and fetch.go. Shapes are broader than the
// plain DOM definitions — `new Audio(src)` accepts streams /
// buffers on the brainkit side, and the WebSocket constructor
// carries Node `ws`'s custom-header + EventEmitter surface
// alongside WHATWG.

/** Web-standard Audio — backed by the configured `audio.Sink`. */
declare class Audio {
  /**
   * @param src URL string, filesystem path, Buffer, Uint8Array,
   *            Blob, Node Readable, or Web ReadableStream.
   */
  constructor(src?: string | any);

  src: string | any;
  paused: boolean;
  ended: boolean;
  currentTime: number;
  duration: number;
  volume: number;
  muted: boolean;
  loop: boolean;
  autoplay: boolean;
  preload: "none" | "metadata" | "auto";

  onplay: ((ev?: any) => void) | null;
  onpause: ((ev?: any) => void) | null;
  onended: ((ev?: any) => void) | null;
  onerror: ((ev?: any) => void) | null;

  /** Resolves when playback ends. Rejects on sink error. */
  play(): Promise<void>;
  pause(): void;
  load(): void;
  canPlayType(type: string): "" | "maybe" | "probably";

  addEventListener(event: "play" | "pause" | "ended" | "error", listener: (ev?: any) => void): void;
  removeEventListener(event: string, listener: (ev?: any) => void): void;
}

/**
 * Client WebSocket — combined WHATWG + Node `ws` surface.
 * Supports custom headers via the 3-arg constructor
 * (`new WebSocket(url, protocols, { headers })`) for
 * Authorization + OpenAI-Beta handshake scenarios.
 */
declare class WebSocket {
  constructor(
    url: string,
    protocols?: string | string[],
    options?: { headers?: Record<string, string>; [key: string]: any },
  );

  readonly url: string;
  readonly protocol: string;
  readonly extensions: string;
  readonly readyState: number;
  readonly bufferedAmount: number;
  binaryType: "nodebuffer" | "arraybuffer" | "blob" | "fragments";

  onopen: ((ev?: any) => void) | null;
  onmessage: ((ev: { data: any; type: "message" }) => void) | null;
  onerror: ((ev?: any) => void) | null;
  onclose: ((ev: { code: number; reason: string; wasClean: boolean }) => void) | null;

  static readonly CONNECTING: 0;
  static readonly OPEN: 1;
  static readonly CLOSING: 2;
  static readonly CLOSED: 3;
  readonly CONNECTING: 0;
  readonly OPEN: 1;
  readonly CLOSING: 2;
  readonly CLOSED: 3;

  // WHATWG.
  send(data: string | ArrayBufferLike | ArrayBufferView | Blob, cb?: (err?: Error) => void): void;
  close(code?: number, reason?: string): void;
  addEventListener(event: "open" | "message" | "error" | "close", listener: (ev?: any) => void): void;
  removeEventListener(event: string, listener: (ev?: any) => void): void;
  dispatchEvent(event: any): boolean;

  // Node `ws` extensions — EventEmitter surface + terminate().
  on(event: "open", listener: () => void): this;
  on(event: "message", listener: (data: any) => void): this;
  on(event: "error", listener: (err: Error) => void): this;
  on(event: "close", listener: (code: number, reason: string) => void): this;
  on(event: string, listener: (...args: any[]) => void): this;
  once(event: string, listener: (...args: any[]) => void): this;
  off(event: string, listener: (...args: any[]) => void): this;
  removeListener(event: string, listener: (...args: any[]) => void): this;
  emit(event: string, ...args: any[]): boolean;
  terminate(): void;
}

/** FormData polyfill. Multipart/form-data body for fetch uploads. */
declare class FormData {
  constructor();
  append(name: string, value: string | Blob, filename?: string): void;
  set(name: string, value: string | Blob, filename?: string): void;
  delete(name: string): void;
  has(name: string): boolean;
  get(name: string): string | Blob | null;
  getAll(name: string): Array<string | Blob>;
  entries(): IterableIterator<[string, string | Blob]>;
  keys(): IterableIterator<string>;
  values(): IterableIterator<string | Blob>;
  forEach(fn: (value: string | Blob, name: string, parent: FormData) => void): void;
  [Symbol.iterator](): IterableIterator<[string, string | Blob]>;
}

/** Blob polyfill — carries bytes verbatim for FormData / fetch. */
declare class Blob {
  constructor(parts?: any[], options?: { type?: string });
  readonly size: number;
  readonly type: string;
  arrayBuffer(): Promise<ArrayBuffer>;
  text(): Promise<string>;
  stream(): any;
  slice(start?: number, end?: number, contentType?: string): Blob;
}

/** File extends Blob with a filename + last-modified timestamp. */
declare class File extends Blob {
  constructor(parts: any[], name: string, options?: { type?: string; lastModified?: number });
  readonly name: string;
  readonly lastModified: number;
}

/**
 * Brainkit's typed error class — thrown from every bus helper
 * (`bus.call`, `msg.reply({ ok: false, ... })`) and from the
 * runtime dispatch layer. Extends `Error` with `code` + `details`
 * as first-class constructor parameters.
 *
 * @example
 * ```ts
 * try {
 *   await bus.call("some.topic", data);
 * } catch (err) {
 *   if (err instanceof BrainkitError && err.code === "TIMEOUT") {
 *     // handle timeout
 *   }
 * }
 * ```
 */
declare class BrainkitError extends Error {
  constructor(message: string, code?: string, details?: Record<string, unknown>);
  readonly name: "BrainkitError";
  readonly code: string;
  readonly details: Record<string, unknown>;
}

// ── Timers + console (jsbridge/timers.go + console.go) ────────

declare const setTimeout: (cb: (...args: any[]) => void, ms?: number, ...args: any[]) => number;
declare const setInterval: (cb: (...args: any[]) => void, ms?: number, ...args: any[]) => number;
declare const clearTimeout: (id?: number) => void;
declare const clearInterval: (id?: number) => void;
declare const setImmediate: (cb: (...args: any[]) => void, ...args: any[]) => number;
declare const clearImmediate: (id?: number) => void;
declare const queueMicrotask: (cb: () => void) => void;

declare const console: {
  log(...args: any[]): void;
  info(...args: any[]): void;
  warn(...args: any[]): void;
  error(...args: any[]): void;
  debug(...args: any[]): void;
  trace(...args: any[]): void;
  dir(obj: any, options?: any): void;
  time(label?: string): void;
  timeEnd(label?: string): void;
  table(data: any, columns?: string[]): void;
  group(...args: any[]): void;
  groupEnd(): void;
  clear(): void;
  count(label?: string): void;
};

// ── Ambient endowments (Compartment-scope, no import needed) ──
//
// kit_runtime.js endows these on every `.ts` deployment's
// Compartment scope alongside the module-style `import`s. Tests
// + fixtures frequently reference them as bare identifiers —
// the declarations below make IDE completion match the real
// runtime. Every name here is also available via
// `import { X } from "kit" | "agent" | "ai"` — type shapes are
// authoritative over there and these are aliases.

// "kit" module — the brainkit-specific surface.
declare const bus: typeof import("kit").bus;
declare const kit: typeof import("kit").kit;
declare const model: typeof import("kit").model;
declare const embeddingModel: typeof import("kit").embeddingModel;
declare const provider: typeof import("kit").provider;
declare const storage: typeof import("kit").storage;
declare const vectorStore: typeof import("kit").vectorStore;
declare const registry: typeof import("kit").registry;
declare const tools: typeof import("kit").tools;
declare const tool: typeof import("kit").tool;
declare const mcp: typeof import("kit").mcp;
declare const output: typeof import("kit").output;
declare const secrets: typeof import("kit").secrets;
declare const generateWithApproval: typeof import("kit").generateWithApproval;
declare const fs: typeof import("kit").fs;

// "agent" (Mastra) — classes + helpers endowed as bare names.
declare const Agent: typeof import("agent").Agent;
declare const createTool: typeof import("agent").createTool;
declare const createWorkflow: typeof import("agent").createWorkflow;
declare const createStep: typeof import("agent").createStep;
declare const Mastra: typeof import("agent").Mastra;
declare const Memory: typeof import("agent").Memory;
declare const MockMemory: typeof import("agent").MockMemory;
declare const InMemoryStore: typeof import("agent").InMemoryStore;
declare const LibSQLStore: typeof import("agent").LibSQLStore;
declare const LibSQLVector: typeof import("agent").LibSQLVector;
declare const UpstashStore: typeof import("agent").UpstashStore;
declare const PostgresStore: typeof import("agent").PostgresStore;
declare const PgVector: typeof import("agent").PgVector;
declare const MongoDBStore: typeof import("agent").MongoDBStore;
declare const MongoDBVector: typeof import("agent").MongoDBVector;
declare const RequestContext: typeof import("agent").RequestContext;
declare const MASTRA_RESOURCE_ID_KEY: typeof import("agent").MASTRA_RESOURCE_ID_KEY;
declare const MASTRA_THREAD_ID_KEY: typeof import("agent").MASTRA_THREAD_ID_KEY;
declare const Workspace: typeof import("agent").Workspace;
declare const LocalFilesystem: typeof import("agent").LocalFilesystem;
declare const LocalSandbox: typeof import("agent").LocalSandbox;

// RAG
declare const MDocument: typeof import("agent").MDocument;
declare const GraphRAG: typeof import("agent").GraphRAG;
declare const createVectorQueryTool: typeof import("agent").createVectorQueryTool;
declare const createDocumentChunkerTool: typeof import("agent").createDocumentChunkerTool;
declare const createGraphRAGTool: typeof import("agent").createGraphRAGTool;
declare const rerank: typeof import("agent").rerank;
declare const rerankWithScorer: typeof import("agent").rerankWithScorer;

// Observability
declare const Observability: typeof import("agent").Observability;
declare const DefaultExporter: typeof import("agent").DefaultExporter;
declare const SensitiveDataFilter: typeof import("agent").SensitiveDataFilter;

// Evals / Scorers
declare const createScorer: typeof import("agent").createScorer;
declare const runEvals: typeof import("agent").runEvals;
declare const createCompletenessScorer: typeof import("agent").createCompletenessScorer;
declare const createTextualDifferenceScorer: typeof import("agent").createTextualDifferenceScorer;
declare const createKeywordCoverageScorer: typeof import("agent").createKeywordCoverageScorer;
declare const createContentSimilarityScorer: typeof import("agent").createContentSimilarityScorer;
declare const createToneScorer: typeof import("agent").createToneScorer;
declare const createAnswerRelevancyScorer: typeof import("agent").createAnswerRelevancyScorer;
declare const createAnswerSimilarityScorer: typeof import("agent").createAnswerSimilarityScorer;
declare const createFaithfulnessScorer: typeof import("agent").createFaithfulnessScorer;
declare const createHallucinationScorer: typeof import("agent").createHallucinationScorer;
declare const createBiasScorer: typeof import("agent").createBiasScorer;
declare const createToxicityScorer: typeof import("agent").createToxicityScorer;
declare const createContextPrecisionScorer: typeof import("agent").createContextPrecisionScorer;
declare const createContextRelevanceScorerLLM: typeof import("agent").createContextRelevanceScorerLLM;
declare const createNoiseSensitivityScorerLLM: typeof import("agent").createNoiseSensitivityScorerLLM;
declare const createPromptAlignmentScorerLLM: typeof import("agent").createPromptAlignmentScorerLLM;
declare const createToolCallAccuracyScorerLLM: typeof import("agent").createToolCallAccuracyScorerLLM;

// Processors
declare const ModerationProcessor: typeof import("agent").ModerationProcessor;
declare const PromptInjectionDetector: typeof import("agent").PromptInjectionDetector;
declare const PIIDetector: typeof import("agent").PIIDetector;
declare const SystemPromptScrubber: typeof import("agent").SystemPromptScrubber;
declare const UnicodeNormalizer: typeof import("agent").UnicodeNormalizer;
declare const LanguageDetector: typeof import("agent").LanguageDetector;
declare const TokenLimiterProcessor: typeof import("agent").TokenLimiterProcessor;
declare const BatchPartsProcessor: typeof import("agent").BatchPartsProcessor;
declare const StructuredOutputProcessor: typeof import("agent").StructuredOutputProcessor;
declare const ToolCallFilter: typeof import("agent").ToolCallFilter;
declare const ToolSearchProcessor: typeof import("agent").ToolSearchProcessor;
declare const AgentsMDInjector: typeof import("agent").AgentsMDInjector;
declare const SkillsProcessor: typeof import("agent").SkillsProcessor;
declare const SkillSearchProcessor: typeof import("agent").SkillSearchProcessor;
declare const WorkspaceInstructionsProcessor: typeof import("agent").WorkspaceInstructionsProcessor;

// Voice providers
declare const MastraVoice: typeof import("agent").MastraVoice;
declare const CompositeVoice: typeof import("agent").CompositeVoice;
declare const OpenAIVoice: typeof import("agent").OpenAIVoice;
declare const OpenAIRealtimeVoice: typeof import("agent").OpenAIRealtimeVoice;
declare const AzureVoice: typeof import("agent").AzureVoice;
declare const ElevenLabsVoice: typeof import("agent").ElevenLabsVoice;
declare const CloudflareVoice: typeof import("agent").CloudflareVoice;
declare const DeepgramVoice: typeof import("agent").DeepgramVoice;
declare const PlayAIVoice: typeof import("agent").PlayAIVoice;
declare const SpeechifyVoice: typeof import("agent").SpeechifyVoice;
declare const SarvamVoice: typeof import("agent").SarvamVoice;
declare const MurfVoice: typeof import("agent").MurfVoice;
declare const DefaultVoice: typeof import("agent").DefaultVoice;

// "ai" module — AI SDK helpers as bare identifiers.
declare const generateText: typeof import("ai").generateText;
declare const streamText: typeof import("ai").streamText;
declare const generateObject: typeof import("ai").generateObject;
declare const streamObject: typeof import("ai").streamObject;
declare const embed: typeof import("ai").embed;
declare const embedMany: typeof import("ai").embedMany;
declare const z: typeof import("ai").z;
declare const jsonSchema: typeof import("ai").jsonSchema;
declare const wrapLanguageModel: typeof import("ai").wrapLanguageModel;
declare const extractReasoningMiddleware: typeof import("ai").extractReasoningMiddleware;
declare const defaultSettingsMiddleware: typeof import("ai").defaultSettingsMiddleware;
declare const stepCountIs: typeof import("ai").stepCountIs;
declare const hasToolCall: typeof import("ai").hasToolCall;

declare class GoSocket {
  connect(portOrOpts: number | { host?: string; port?: number; tls?: boolean }, host?: string): this;
  write(data: any, encoding?: string, cb?: (err?: Error) => void): boolean;
  end(data?: any, encoding?: string, cb?: () => void): void;
  destroy(err?: Error): this;
  pipe(dest: any, opts?: { end?: boolean }): any;
  on(event: string, listener: (...args: any[]) => void): this;
  once(event: string, listener: (...args: any[]) => void): this;
  removeListener(event: string, listener: (...args: any[]) => void): this;
  emit(event: string, ...args: any[]): boolean;
  setNoDelay(noDelay?: boolean): this;
  setKeepAlive(enable?: boolean, delay?: number): this;
  setTimeout(ms: number, cb?: () => void): this;
  readonly remoteAddress: string;
  readonly remotePort: number;
}
