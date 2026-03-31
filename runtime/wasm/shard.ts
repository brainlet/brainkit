// runtime/wasm/shard.ts — Shard registration + bus functions.

import { _busOn, _busPublish, _busEmit, _tool, _reply, _setMode } from "./host"

/** Subscribe to a topic pattern. Handler function is called when messages match. Init only. */
export function on(topic: string, handlerFuncName: string): void {
    _busOn(topic, handlerFuncName)
}

/** Publish to bus with replyTo. Callback receives the reply. */
export function publish(topic: string, payload: string, callbackFuncName: string): void {
    _busPublish(topic, payload, callbackFuncName)
}

/** Fire-and-forget bus publish. No replyTo. */
export function emit(topic: string, payload: string): void {
    _busEmit(topic, payload)
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
