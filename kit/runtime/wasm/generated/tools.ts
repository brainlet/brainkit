// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: tools

export class ToolCallMsg {
    name: string
    input: string

    constructor(name: string, input: string) {
        this.name = name
        this.input = input
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        if (this.input.length > 0) obj.set("input", JSONValue.parse(this.input))
        return obj.toString()
    }
}

export class ToolListMsg {
    namespace: string

    constructor(namespace: string) {
        this.namespace = namespace
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("namespace", this.namespace)
        return obj.toString()
    }
}

export class ToolResolveMsg {
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

export class ToolCallResp {
    result: string
    error: string

    constructor() {
        this.result = ""
        this.error = ""
    }

    static parse(json: string): ToolCallResp {
        let resp = new ToolCallResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("result")) resp.result = obj.get("result").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class ToolListResp {
    tools: string
    error: string

    constructor() {
        this.tools = ""
        this.error = ""
    }

    static parse(json: string): ToolListResp {
        let resp = new ToolListResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("tools")) resp.tools = obj.get("tools").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class ToolResolveResp {
    name: string
    shortName: string
    description: string
    inputSchema: string
    error: string

    constructor() {
        this.name = ""
        this.shortName = ""
        this.description = ""
        this.inputSchema = ""
        this.error = ""
    }

    static parse(json: string): ToolResolveResp {
        let resp = new ToolResolveResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("name")) resp.name = obj.getString("name")
            if (obj.has("shortName")) resp.shortName = obj.getString("shortName")
            if (obj.has("description")) resp.description = obj.getString("description")
            if (obj.has("inputSchema")) resp.inputSchema = obj.get("inputSchema").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace tools {
    export function call(msg: ToolCallMsg, callback: string): void {
        _invokeAsync("tools.call", msg.toJSON(), callback)
    }

    export function list(msg: ToolListMsg, callback: string): void {
        _invokeAsync("tools.list", msg.toJSON(), callback)
    }

    export function resolve(msg: ToolResolveMsg, callback: string): void {
        _invokeAsync("tools.resolve", msg.toJSON(), callback)
    }

}
