// runtime/wasm/host.ts — raw host function bindings.
// These are NOT exported to developers. api.ts wraps them with typed interfaces.

@external("host", "log")
export declare function _host_log(message: string, level: i32): void;

@external("host", "call_tool")
export declare function _host_call_tool(name: string, argsJSON: string): string;

@external("host", "call_agent")
export declare function _host_call_agent(name: string, prompt: string): string;

@external("host", "get_state")
export declare function _host_get_state(key: string): string;

@external("host", "set_state")
export declare function _host_set_state(key: string, value: string): void;

@external("host", "has_state")
export declare function _host_has_state(key: string): i32;

@external("host", "bus_send")
export declare function _host_bus_send(topic: string, payloadJSON: string): void;
