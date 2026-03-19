// runtime/wasm/state.ts — State management functions.

import { _getState, _setState, _hasState } from "./host"

/** Get a value from per-execution state. Returns "" if not found. */
export function getState(key: string): string {
    return _getState(key)
}

/** Set a value in per-execution state. */
export function setState(key: string, value: string): void {
    _setState(key, value)
}

/** Check if a key exists in state. */
export function hasState(key: string): bool {
    return _hasState(key) != 0
}
