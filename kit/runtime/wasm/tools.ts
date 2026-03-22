// runtime/wasm/tools.ts — Tools domain typed messages + namespace functions.

import { _invokeAsync } from "./host"

export class ToolCallMsg {
    name: string
    input: string // JSON

    constructor(name: string, input: string) {
        this.name = name
        this.input = input
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.set("input", JSONValue.parse(this.input))
        return obj.toString()
    }
}

export namespace tools {
    export function call(msg: ToolCallMsg, callback: string): void {
        _invokeAsync("tools.call", msg.toJSON(), callback)
    }
}
