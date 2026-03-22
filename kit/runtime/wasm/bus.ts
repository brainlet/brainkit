// runtime/wasm/bus.ts — Raw bus primitives for custom topics/events.

import { _send } from "./host"
import { BusMsg } from "./types"

export namespace bus {
    /** Send a typed custom event (fire-and-forget). */
    export function send(msg: BusMsg): void {
        _send(msg.topic(), msg.toJSON())
    }

    /** Send raw topic + payload (fire-and-forget). For advanced use. */
    export function sendRaw(topic: string, payload: string): void {
        _send(topic, payload)
    }
}
