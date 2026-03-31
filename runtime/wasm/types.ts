// runtime/wasm/types.ts — Base types for the brainkit WASM module.

/** BusMsg interface — all typed messages must implement this. */
export interface BusMsg {
    topic(): string
    toJSON(): string
}
