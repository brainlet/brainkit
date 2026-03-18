// wasm_host.ts — AssemblyScript host function declarations.
// Include in your AS module to call Kit host functions from WASM.
//
// These @external declarations map to the "host" wazero module registered
// by brainkit's WASMService. Available when running via wasm.run().
//
// Strings are passed natively — AS passes string pointers, the Go host
// reads them using the AS object header layout (rtSize at -4, UTF-16LE payload).
// Return strings are allocated by the host via __new and returned as pointers.
//
// Usage:
//   import { log, callAgent, callTool, getState, setState, busSend } from "./wasm_host";
//
//   export function run(): i32 {
//     log("Starting automation...");
//     const result = callAgent("coder", "write a hello world function");
//     setState("lastResult", result);
//     busSend("automation.complete", '{"status":"done"}');
//     return 0;
//   }

// ── Host function imports ──────────────────────────────────────────
// Strings are passed as-is. AS handles the pointer conversion automatically.

/** Log a message. level: 0=debug, 1=info, 2=warn, 3=error */
@external("host", "log")
export declare function log(message: string, level: i32): void;

/** Call a registered tool. Returns the result as a JSON string. */
@external("host", "call_tool")
export declare function callTool(name: string, argsJSON: string): string;

/** Call a named agent. Returns the agent's text response. */
@external("host", "call_agent")
export declare function callAgent(name: string, prompt: string): string;

/** Get a value from per-execution state. Returns "" if not found. */
@external("host", "get_state")
export declare function getState(key: string): string;

/** Set a value in per-execution state. */
@external("host", "set_state")
export declare function setState(key: string, value: string): void;

/** Publish a message on the Kit bus. */
@external("host", "bus_send")
export declare function busSend(topic: string, payloadJSON: string): void;
