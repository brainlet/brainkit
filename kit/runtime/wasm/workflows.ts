// runtime/wasm/workflows.ts — Workflows domain typed messages + namespace functions.

import { _invokeAsync } from "./host"

export class WorkflowRunMsg {
    name: string
    inputJSON: string

    constructor(name: string, inputJSON: string) {
        this.name = name
        this.inputJSON = inputJSON
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.set("input", JSONValue.parse(this.inputJSON))
        return obj.toString()
    }
}

export namespace workflows {
    export function run(msg: WorkflowRunMsg, callback: string): void {
        _invokeAsync("workflows.run", msg.toJSON(), callback)
    }
}
