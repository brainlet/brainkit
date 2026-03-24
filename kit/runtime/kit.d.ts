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
    publish(topic: string, data?: any): { replyTo: string; correlationId: string };
    /** Fire-and-forget. No replyTo. */
    emit(topic: string, data?: any): void;
    /** Listen on any absolute topic. */
    subscribe(topic: string, handler: (msg: BusMessage) => void | Promise<void>): string;
    /** Listen on deployment mailbox (ts.<source>.<localTopic>). */
    on(localTopic: string, handler: (msg: BusMessage) => void | Promise<void>): string;
    /** Remove a subscription. */
    unsubscribe(subId: string): void;
  };

  interface BusMessage {
    payload: any;
    replyTo: string;
    correlationId: string;
    topic: string;
    callerId: string;
    /** Publish final response to replyTo (done=true). */
    reply(data: any): void;
    /** Publish intermediate chunk to replyTo (done=false). For streaming. */
    send(data: any): void;
  }

  // ── Resource Registration ────────────────────────────────────

  export const kit: {
    /** Register a resource for discovery + teardown lifecycle. */
    register(type: "agent" | "tool" | "workflow" | "memory", name: string, ref: any): void;
    /** Manually unregister a resource. */
    unregister(type: string, name: string): void;
    /** List resources in this deployment. */
    list(type?: string): Array<{ type: string; id: string; name: string; source: string; createdAt: number }>;
    /** This deployment's source name. */
    readonly source: string;
    /** This Kit's namespace. */
    readonly namespace: string;
    /** This Kit's caller ID. */
    readonly callerId: string;
  };

  // ── Model Resolution ─────────────────────────────────────────

  /** Resolve a model from the provider registry. Returns an AI SDK model instance. */
  export function model(provider: string, modelId: string): any;

  /** Resolve a provider factory from the registry. */
  export function provider(name: string): any;

  // ── Provider Registry ────────────────────────────────────────

  export const registry: {
    has(category: "provider" | "vectorStore" | "storage", name: string): boolean;
    list(category: string): any[];
    resolve(category: string, name: string): any;
    register(category: string, name: string, config: any): void;
    unregister(category: string, name: string): void;
  };

  // ── Storage / Vector Resolution ──────────────────────────────

  /** Resolve a Mastra storage instance from the registry. */
  export function storage(name: string): any;

  /** Resolve a Mastra vector store instance from the registry. */
  export function vectorStore(name: string): any;

  // ── Tool Registry ────────────────────────────────────────────

  export const tools: {
    /** Call a registered tool by name. */
    call(name: string, input?: any): Promise<any>;
    /** List all tools visible to this Kit. */
    list(namespace?: string): any[];
    /** Resolve a tool by name — returns tool info. */
    resolve(name: string): any;
  };

  // ── Filesystem ───────────────────────────────────────────────

  export const fs: {
    read(path: string): Promise<{ data: string }>;
    write(path: string, data: string): Promise<{ ok: boolean }>;
    list(path?: string, pattern?: string): Promise<{ files: Array<{ name: string; size: number; isDir: boolean }> }>;
    stat(path: string): Promise<{ size: number; isDir: boolean; modTime: string }>;
    delete(path: string): Promise<{ ok: boolean }>;
    mkdir(path: string): Promise<{ ok: boolean }>;
  };

  // ── MCP ──────────────────────────────────────────────────────

  export const mcp: {
    listTools(server?: string): any[];
    callTool(server: string, tool: string, args?: any): Promise<any>;
  };

  // ── Output ───────────────────────────────────────────────────

  /** Set the module's output value. Passes results back to Go. */
  export function output(value: any): void;
}
