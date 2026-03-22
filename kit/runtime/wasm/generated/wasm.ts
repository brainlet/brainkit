// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: wasm

export class WasmCompileMsg {
    source: string
    options: string

    constructor(source: string, options: string) {
        this.source = source
        this.options = options
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        if (this.options.length > 0) obj.set("options", JSONValue.parse(this.options))
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

export class WasmDescribeMsg {
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

export class WasmGetMsg {
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

export class WasmListMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class WasmRemoveMsg {
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

export class WasmRunMsg {
    moduleId: string
    input: string

    constructor(moduleId: string, input: string) {
        this.moduleId = moduleId
        this.input = input
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("moduleId", this.moduleId)
        if (this.input.length > 0) obj.set("input", JSONValue.parse(this.input))
        return obj.toString()
    }
}

export class WasmUndeployMsg {
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

export class WasmCompileResp {
    moduleId: string
    name: string
    size: i32
    exports: string
    text: string
    error: string

    constructor() {
        this.moduleId = ""
        this.name = ""
        this.size = 0
        this.exports = ""
        this.text = ""
        this.error = ""
    }

    static parse(json: string): WasmCompileResp {
        let resp = new WasmCompileResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("moduleId")) resp.moduleId = obj.getString("moduleId")
            if (obj.has("name")) resp.name = obj.getString("name")
            if (obj.has("size")) resp.size = obj.getInt("size")
            if (obj.has("exports")) resp.exports = obj.get("exports").toString()
            if (obj.has("text")) resp.text = obj.getString("text")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmDeployResp {
    module: string
    mode: string
    handlers: string
    error: string

    constructor() {
        this.module = ""
        this.mode = ""
        this.handlers = ""
        this.error = ""
    }

    static parse(json: string): WasmDeployResp {
        let resp = new WasmDeployResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("module")) resp.module = obj.getString("module")
            if (obj.has("mode")) resp.mode = obj.getString("mode")
            if (obj.has("handlers")) resp.handlers = obj.get("handlers").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmDescribeResp {
    module: string
    mode: string
    handlers: string
    error: string

    constructor() {
        this.module = ""
        this.mode = ""
        this.handlers = ""
        this.error = ""
    }

    static parse(json: string): WasmDescribeResp {
        let resp = new WasmDescribeResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("module")) resp.module = obj.getString("module")
            if (obj.has("mode")) resp.mode = obj.getString("mode")
            if (obj.has("handlers")) resp.handlers = obj.get("handlers").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmGetResp {
    module: string
    error: string

    constructor() {
        this.module = ""
        this.error = ""
    }

    static parse(json: string): WasmGetResp {
        let resp = new WasmGetResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("module")) resp.module = obj.get("module").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmListResp {
    modules: string
    error: string

    constructor() {
        this.modules = ""
        this.error = ""
    }

    static parse(json: string): WasmListResp {
        let resp = new WasmListResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("modules")) resp.modules = obj.get("modules").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmRemoveResp {
    removed: bool
    error: string

    constructor() {
        this.removed = false
        this.error = ""
    }

    static parse(json: string): WasmRemoveResp {
        let resp = new WasmRemoveResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("removed")) resp.removed = obj.getBool("removed")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmRunResp {
    exitCode: i32
    value: string
    error: string

    constructor() {
        this.exitCode = 0
        this.value = ""
        this.error = ""
    }

    static parse(json: string): WasmRunResp {
        let resp = new WasmRunResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("exitCode")) resp.exitCode = obj.getInt("exitCode")
            if (obj.has("value")) resp.value = obj.get("value").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WasmUndeployResp {
    undeployed: bool
    error: string

    constructor() {
        this.undeployed = false
        this.error = ""
    }

    static parse(json: string): WasmUndeployResp {
        let resp = new WasmUndeployResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("undeployed")) resp.undeployed = obj.getBool("undeployed")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace wasm_ops {
    export function compile(msg: WasmCompileMsg, callback: string): void {
        _invokeAsync("wasm.compile", msg.toJSON(), callback)
    }

    export function deploy(msg: WasmDeployMsg, callback: string): void {
        _invokeAsync("wasm.deploy", msg.toJSON(), callback)
    }

    export function describe(msg: WasmDescribeMsg, callback: string): void {
        _invokeAsync("wasm.describe", msg.toJSON(), callback)
    }

    export function get(msg: WasmGetMsg, callback: string): void {
        _invokeAsync("wasm.get", msg.toJSON(), callback)
    }

    export function list(msg: WasmListMsg, callback: string): void {
        _invokeAsync("wasm.list", msg.toJSON(), callback)
    }

    export function remove(msg: WasmRemoveMsg, callback: string): void {
        _invokeAsync("wasm.remove", msg.toJSON(), callback)
    }

    export function run(msg: WasmRunMsg, callback: string): void {
        _invokeAsync("wasm.run", msg.toJSON(), callback)
    }

    export function undeploy(msg: WasmUndeployMsg, callback: string): void {
        _invokeAsync("wasm.undeploy", msg.toJSON(), callback)
    }

}
