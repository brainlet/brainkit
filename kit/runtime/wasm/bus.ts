// runtime/wasm/bus.ts — Raw bus primitives for custom topics/events.

import { _send, _askAsync } from "./host"
import { BusMsg } from "./types"

export namespace bus {
    /** Send a typed custom event (fire-and-forget). */
    export function send(msg: BusMsg): void {
        _send(msg.topic(), msg.toJSON())
    }

    /** Async ask with a typed custom message. Callback function called when response arrives. */
    export function askAsync(msg: BusMsg, callbackFuncName: string): void {
        _askAsync(msg.topic(), msg.toJSON(), callbackFuncName)
    }

    /** Send raw topic + payload (fire-and-forget). For advanced use. */
    export function sendRaw(topic: string, payload: string): void {
        _send(topic, payload)
    }

    /** Async ask with raw topic + payload. For advanced use. */
    export function askAsyncRaw(topic: string, payload: string, callbackFuncName: string): void {
        _askAsync(topic, payload, callbackFuncName)
    }
}
