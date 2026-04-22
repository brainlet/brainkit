/**
 * "kit" module — brainkit infrastructure.
 * This is the only module brainkit fully owns.
 *
 * AI SDK comes from "ai", Mastra from "agent". This module provides
 * bus messaging, model resolution, tool registry, filesystem, and MCP.
 */
declare module "kit" {

  // ── Bus (messaging) ──────────────────────────────────────────

  export const bus: {
    /** Send + expect reply. Returns routing info. */
    publish(topic: string, data?: unknown): { replyTo: string; correlationId: string };
    /** Fire-and-forget. No replyTo. */
    emit(topic: string, data?: unknown): void;
    /** Listen on any absolute topic. */
    subscribe(topic: string, handler: (msg: BusMessage) => void | Promise<void>): string;
    /** Listen on deployment mailbox (ts.<source>.<localTopic>). */
    on(localTopic: string, handler: (msg: BusMessage) => void | Promise<void>): string;
    /** Remove a subscription. */
    unsubscribe(subId: string): void;
    /** Send to a deployed .ts service by name.
     *  Resolves "my-agent.ts" + "ask" → publishes to ts.my-agent.ask */
    sendTo(service: string, topic: string, data?: unknown): { replyTo: string; correlationId: string };
    /**
     * Send a request-reply command and await the terminal envelope.
     * Throws BrainkitError on the remote handler's ok=false reply;
     * throws a TIMEOUT BrainkitError if timeoutMs elapses first.
     * timeoutMs is REQUIRED — mirrors Go's deadline rule.
     *
     * @example
     *   const reply = await bus.call("ts.my-svc.chat", { text: "hi" }, { timeoutMs: 5000 });
     */
    call<T = any>(topic: string, data?: unknown, opts?: { timeoutMs: number }): Promise<T>;
    /** Cross-kit variant of call(): routes the request to a different namespace. */
    callTo<T = any>(namespace: string, topic: string, data?: unknown, opts?: { timeoutMs: number }): Promise<T>;
    /**
     * Subscribe to cancel signals for an in-flight Call identified by
     * correlationId. The handler fires when the upstream caller
     * cancels (ctx done, timeout, explicit abort). Returns an
     * unsubscribe function.
     *
     * @example
     *   bus.on("expensive", async (msg) => {
     *     const unsub = bus.onCancel(msg.correlationId, () => abort());
     *     try { return await run(); }
     *     finally { unsub(); }
     *   });
     */
    onCancel(correlationId: string, handler: (evt: any) => void): () => void;
    /**
     * Build an AbortController wired to the cancel signal for msg's
     * correlationId. Pass the returned signal to fetch() / any
     * AbortController-aware API and call cleanup before returning.
     *
     * @example
     *   bus.on("long-request", async (msg) => {
     *     const { signal, cleanup } = bus.withCancelController(msg);
     *     try {
     *       const res = await fetch(url, { signal });
     *       msg.reply(await res.json());
     *     } finally { cleanup(); }
     *   });
     */
    withCancelController(msg: BusMessage): { signal: AbortSignal; cleanup: () => void };
    /**
     * Schedule a message. `expression` is a cron spec
     * ("*\/5 * * * *"), a human-readable delay ("in 1h",
     * "every day at 9am"), or any string the scheduler
     * module recognizes. Returns the schedule id
     * synchronously — cancel via `bus.unschedule(id)`.
     *
     * Inside a deployed `.ts`, `topic` is auto-prefixed with the
     * deployment's namespace (`ts.<source>.<topic>`). Pass a topic
     * that matches a `bus.on(...)` handler in the same deployment —
     * NOT the full namespaced topic. Passing the full topic
     * double-prefixes and the scheduled message silently misses.
     *
     * @example
     *     const id = bus.schedule("in 1h", "my.topic", { payload: "x" });
     *     bus.unschedule(id);
     */
    schedule(expression: string, topic: string, data?: unknown): string;
    /** Cancel a scheduled publish by id. */
    unschedule(scheduleId: string): void;
  };

  export interface BusMessage {
    payload: any;
    replyTo: string;
    correlationId: string;
    topic: string;
    callerId: string;
    /** Publish final response to replyTo (done=true). */
    reply(data: any): void;
    /** Publish intermediate chunk to replyTo (done=false). For streaming. */
    send(data: any): void;
    /** Register a cancel callback on this message's correlationId.
     *  Fires when the caller signals cancellation via the bus; returns
     *  an unsubscribe function. Only present on the per-deployment
     *  wrapped message (not raw SubscribeRaw messages). */
    onCancel?(handler: (evt: any) => void): () => void;
    /** Stream helpers for partial responses. */
    stream?: {
      text(chunk: string): void;
      /** @param value Progress fraction in `[0, 1]`. The gateway/CLI renders it as a percentage — passing `10` renders as `[1000%]`. */
      progress(value: number, message?: string): void;
      object(partial: any): void;
      event(name: string, data?: any): void;
      error(message: string): void;
      end(): void;
    };
  }

  // ── Resource Registration ────────────────────────────────────

  export const kit: {
    /** Register a resource for discovery + teardown lifecycle. */
    register(type: "agent" | "tool" | "workflow" | "memory", name: string, ref: unknown): void;
    /** Manually unregister a resource. */
    unregister(type: string, name: string): void;
    /** List resources in this deployment. */
    list(type?: string): ResourceEntry[];
    /** This deployment's source name. */
    readonly source: string;
    /** This Kit's namespace. */
    readonly namespace: string;
    /** This Kit's caller ID. */
    readonly callerId: string;
  };

  export interface ResourceEntry {
    type: string;
    id: string;
    name: string;
    source: string;
    createdAt: number;
  }

  // ── Model Resolution ─────────────────────────────────────────

  /** Resolve a language model from the provider registry. For generateText, streamText, Agent. */
  export function model(provider: string, modelId: string): any;

  /** Resolve an embedding model from the provider registry. For embed, embedMany. */
  export function embeddingModel(provider: string, modelId: string): any;

  /** Resolve a provider factory from the registry. */
  export function provider(name: string): ProviderFactory;

  /** Provider factory — call with model ID to get a LanguageModel. */
  export interface ProviderFactory {
    (modelId: string): import("ai").LanguageModel;
  }

  // ── Provider Registry ────────────────────────────────────────

  export const registry: {
    has(category: "provider" | "vectorStore" | "storage", name: string): boolean;
    list(category: string): RegistryEntry[];
    resolve(category: string, name: string): RegistryConfig | null;
    register(category: string, name: string, config: Record<string, unknown>): void;
    unregister(category: string, name: string): void;
  };

  export interface RegistryEntry {
    name: string;
    type: string;
    healthy: boolean;
    lastProbed: string;
    lastError: string;
  }

  export interface RegistryConfig {
    type: string;
    name: string;
    config: Record<string, unknown>;
  }

  // ── Storage / Vector Resolution ──────────────────────────────

  /** Resolve a Mastra storage instance from the registry. */
  export function storage(name: string): import("agent").StorageInstance;

  /** Resolve a Mastra vector store instance from the registry. */
  export function vectorStore(name: string): import("agent").VectorStoreInstance;

  // ── Tool Registry ────────────────────────────────────────────

  export const tools: {
    /** Call a registered tool by name. */
    call(name: string, input?: Record<string, unknown>): Promise<unknown>;
    /** List all tools visible to this Kit. */
    list(namespace?: string): ToolInfo[];
    /** Resolve a tool by name — returns tool info. */
    resolve(name: string): ToolResolveResult;
  };

  export interface ToolInfo {
    name: string;
    shortName: string;
    namespace: string;
    description: string;
  }

  export interface ToolResolveResult {
    name: string;
    shortName: string;
    description: string;
    inputSchema?: Record<string, unknown>;
  }

  // ── Filesystem ───────────────────────────────────────────────

  export const fs: {
    readFile(path: string, encoding?: string): Promise<string | Uint8Array>;
    writeFile(path: string, data: string | Uint8Array, options?: { encoding?: string; mode?: number }): Promise<void>;
    appendFile(path: string, data: string | Uint8Array, options?: { encoding?: string }): Promise<void>;
    readdir(path: string, options?: { withFileTypes?: boolean }): Promise<string[] | any[]>;
    stat(path: string): Promise<{ size: number; mtime: Date; isFile(): boolean; isDirectory(): boolean }>;
    lstat(path: string): Promise<{ size: number; mtime: Date; isFile(): boolean; isDirectory(): boolean; isSymbolicLink(): boolean }>;
    access(path: string, mode?: number): Promise<void>;
    mkdir(path: string, options?: { recursive?: boolean }): Promise<void>;
    rm(path: string, options?: { recursive?: boolean; force?: boolean }): Promise<void>;
    unlink(path: string): Promise<void>;
    rename(oldPath: string, newPath: string): Promise<void>;
    copyFile(src: string, dest: string, flags?: number): Promise<void>;
    readFileSync(path: string, encoding?: string): string | Uint8Array;
    writeFileSync(path: string, data: string | Uint8Array, options?: { encoding?: string; mode?: number }): void;
    readdirSync(path: string, options?: { withFileTypes?: boolean }): string[] | any[];
    statSync(path: string): { size: number; mtime: Date; isFile(): boolean; isDirectory(): boolean };
    lstatSync(path: string): { size: number; mtime: Date; isFile(): boolean; isDirectory(): boolean; isSymbolicLink(): boolean };
    existsSync(path: string): boolean;
    mkdirSync(path: string, options?: { recursive?: boolean }): void;
    unlinkSync(path: string): void;
    rmSync(path: string, options?: { recursive?: boolean; force?: boolean }): void;
    promises: {
      readFile(path: string, encoding?: string): Promise<string | Uint8Array>;
      writeFile(path: string, data: string | Uint8Array, options?: { encoding?: string; mode?: number }): Promise<void>;
      readdir(path: string, options?: { withFileTypes?: boolean }): Promise<string[] | any[]>;
      stat(path: string): Promise<{ size: number; mtime: Date; isFile(): boolean; isDirectory(): boolean }>;
      mkdir(path: string, options?: { recursive?: boolean }): Promise<void>;
      unlink(path: string): Promise<void>;
      rm(path: string, options?: { recursive?: boolean; force?: boolean }): Promise<void>;
      rename(oldPath: string, newPath: string): Promise<void>;
      copyFile(src: string, dest: string, flags?: number): Promise<void>;
      access(path: string, mode?: number): Promise<void>;
      [key: string]: any;
    };
    [key: string]: any;
  };

  export interface FileInfo {
    name: string;
    size: number;
    isDir: boolean;
  }

  // ── MCP ──────────────────────────────────────────────────────

  export const mcp: {
    listTools(server?: string): McpToolInfo[];
    callTool(server: string, tool: string, args?: Record<string, unknown>): Promise<unknown>;
  };

  export interface McpToolInfo {
    name: string;
    server: string;
    description: string;
  }

  // ── Tool Resolution ──────────────────────────────────────────

  /** Resolve a registered tool by name. Returns a tool reference usable with Agent. */
  export function tool(name: string): any;

  // ── Output ───────────────────────────────────────────────────

  /** Set the module's output value. Passes results back to Go. */
  export function output(value: unknown): void;

  // ── Secrets ─────────────────────────────────────────────────

  /**
   * Secret vault — encrypted key/value store backed by
   * `Kit.Secrets()` on the Go side. Names must be registered
   * via `kit.Secrets().Set(...)` or pre-seeded at deploy time.
   * Returns empty string when the name isn't set.
   */
  export const secrets: {
    get(name: string): string;
  };

  // ── HITL (bus-based approval) ───────────────────────────────

  /**
   * Generate with bus-based tool approval (HITL).
   *
   * Thin layer over Agent.generate that routes tool approval through the bus.
   * Any surface (Go, .ts, plugin, gateway) can approve or decline by subscribing
   * to the approvalTopic and calling msg.reply({ approved: true/false }).
   *
   * @example
   * ```ts
   * // .ts service
   * const result = await generateWithApproval(agent, "Delete record X", {
   *   approvalTopic: "approvals.pending",
   *   timeout: 30000,
   * });
   *
   * // Go/plugin/gateway — subscribes and approves
   * bus.subscribe("approvals.pending", (msg) => {
   *   console.log("Tool:", msg.payload.toolName, "Args:", msg.payload.args);
   *   msg.reply({ approved: true });
   * });
   * ```
   */
  export function generateWithApproval(
    agent: import("agent").Agent,
    promptOrMessages: string | import("agent").Message[],
    options: {
      /** Bus topic to publish approval requests to. Required. */
      approvalTopic: string;
      /** Timeout in ms before auto-declining. @default 30000 */
      timeout?: number;
      /** Max suspend/resume cycles before throwing. Models that retry after a decline chain multiple approval cycles within one call. @default 8 */
      maxApprovalCycles?: number;
      /** Memory options (thread, resource). */
      memory?: { thread?: string | { id: string }; resource?: string };
      /** Any other AgentCallOptions. */
      [key: string]: any;
    },
  ): Promise<import("agent").AgentResult>;

  // ── Gateway Route Topics (when gateway module is enabled) ────
  //
  // Deployed packages can register HTTP routes dynamically via bus:
  //
  //   bus.publish("gateway.http.route.add", {
  //     method: "POST",
  //     path: "/api/my-endpoint",
  //     topic: "ts.my-pkg.handle-request",
  //     type: "handle",            // "handle" or "webhook"
  //   });
  //
  //   bus.publish("gateway.http.route.remove", {
  //     method: "POST",
  //     path: "/api/my-endpoint",
  //   });
  //
  //   bus.call("gateway.http.route.list", null, { timeoutMs: 5000 });
  //
  // Supported route types: "handle" (request → bus → response),
  // "webhook" (fire-and-forget, returns 202). SSE/WS route types
  // are Go-only via gw.HandleStream() / gw.HandleWebSocket().
}
