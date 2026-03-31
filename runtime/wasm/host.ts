// runtime/wasm/host.ts — Raw host function bindings (INTERNAL).
// Developers never import this file directly. Namespace files use these.

@external("host", "bus_emit")
export declare function _busEmit(topic: string, payload: string): void

@external("host", "bus_publish")
export declare function _busPublish(topic: string, payload: string, callbackFuncName: string): void

@external("host", "bus_on")
export declare function _busOn(topic: string, funcName: string): void

@external("host", "tool")
export declare function _tool(name: string, funcName: string): void

@external("host", "reply")
export declare function _reply(payload: string): void

@external("host", "log")
export declare function _log(message: string, level: i32): void

@external("host", "get_state")
export declare function _getState(key: string): string

@external("host", "set_state")
export declare function _setState(key: string, value: string): void

@external("host", "has_state")
export declare function _hasState(key: string): i32

@external("host", "set_mode")
export declare function _setMode(mode: string): void
