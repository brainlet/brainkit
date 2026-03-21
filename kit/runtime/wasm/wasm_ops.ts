// runtime/wasm/wasm_ops.ts — WASM operations typed messages + namespace functions.

import { _askAsync } from "./host"

export class WasmCompileMsg {
    source: string
    name: string

    constructor(source: string, name: string = "") {
        this.source = source
        this.name = name
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        if (this.name.length > 0) {
            let opts = new JSONObject()
            opts.setString("name", this.name)
            obj.set("options", opts)
        }
        return obj.toString()
    }
}

export class WasmDeployMsg {
    name: string

    constructor(name: string) {
        this.name = name
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        return obj.toString()
    }
}

export namespace wasm_ops {
    export function compile(msg: WasmCompileMsg, callback: string): void {
        _askAsync("wasm.compile", msg.toJSON(), callback)
    }

    export function deploy(msg: WasmDeployMsg, callback: string): void {
        _askAsync("wasm.deploy", msg.toJSON(), callback)
    }
}
