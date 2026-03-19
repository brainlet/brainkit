// runtime/wasm/log.ts — Logging functions.

import { _log } from "./host"

/** Log at info level */
export function log(message: string): void {
    _log(message, 1)
}

/** Log at specific level: 0=debug, 1=info, 2=warn, 3=error */
export function logAt(message: string, level: i32): void {
    _log(message, level)
}

/** Log at debug level */
export function debug(message: string): void {
    _log(message, 0)
}

/** Log at warn level */
export function warn(message: string): void {
    _log(message, 2)
}

/** Log at error level */
export function error(message: string): void {
    _log(message, 3)
}
