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
  };

  interface BusMessage {
    payload: unknown;
    replyTo: string;
    correlationId: string;
    topic: string;
    callerId: string;
    /** Publish final response to replyTo (done=true). */
    reply(data: unknown): void;
    /** Publish intermediate chunk to replyTo (done=false). For streaming. */
    send(data: unknown): void;
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

  interface ResourceEntry {
    type: string;
    id: string;
    name: string;
    source: string;
    createdAt: number;
  }

  // ── Model Resolution ─────────────────────────────────────────

  /** Resolve a model from the provider registry. Returns an AI SDK model instance. */
  export function model(provider: string, modelId: string): any;

  /** Resolve a provider factory from the registry. */
  export function provider(name: string): ProviderFactory;

  /** Provider factory — call with model ID to get a LanguageModel. */
  interface ProviderFactory {
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

  interface RegistryEntry {
    name: string;
    type: string;
    healthy: boolean;
    lastProbed: string;
    lastError: string;
  }

  interface RegistryConfig {
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

  interface ToolInfo {
    name: string;
    shortName: string;
    namespace: string;
    description: string;
  }

  interface ToolResolveResult {
    name: string;
    shortName: string;
    description: string;
    inputSchema?: Record<string, unknown>;
  }

  // ── Filesystem ───────────────────────────────────────────────

  export const fs: {
    read(path: string): Promise<{ data: string }>;
    write(path: string, data: string): Promise<{ ok: boolean }>;
    list(path?: string, pattern?: string): Promise<{ files: FileInfo[] }>;
    stat(path: string): Promise<{ size: number; isDir: boolean; modTime: string }>;
    delete(path: string): Promise<{ ok: boolean }>;
    mkdir(path: string): Promise<{ ok: boolean }>;
  };

  interface FileInfo {
    name: string;
    size: number;
    isDir: boolean;
  }

  // ── MCP ──────────────────────────────────────────────────────

  export const mcp: {
    listTools(server?: string): McpToolInfo[];
    callTool(server: string, tool: string, args?: Record<string, unknown>): Promise<unknown>;
  };

  interface McpToolInfo {
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
}
