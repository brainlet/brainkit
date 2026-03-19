// runtime/wasm/shard.ts — Shard registration functions (init phase only).

import { _on, _tool, _reply, _setMode } from "./host"

/** Subscribe to a topic pattern. Handler function is called when messages match. Init only. */
export function on(topic: string, handlerFuncName: string): void {
    _on(topic, handlerFuncName)
}

/** Register a tool this shard provides. Init only. */
export function tool(name: string, handlerFuncName: string): void {
    _tool(name, handlerFuncName)
}

/** Reply to the current inbound message. */
export function reply(payload: string): void {
    _reply(payload)
}

/** Set shard execution mode: "stateless" or "persistent". Init only. */
export function setMode(mode: string): void {
    _setMode(mode)
}
