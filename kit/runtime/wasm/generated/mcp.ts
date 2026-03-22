// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: mcp

export class McpCallToolMsg {
    server: string
    tool: string
    args: string

    constructor(server: string, tool: string, args: string) {
        this.server = server
        this.tool = tool
        this.args = args
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("server", this.server)
        obj.setString("tool", this.tool)
        if (this.args.length > 0) obj.set("args", JSONValue.parse(this.args))
        return obj.toString()
    }
}

export class McpListToolsMsg {
    server: string

    constructor(server: string) {
        this.server = server
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("server", this.server)
        return obj.toString()
    }
}

export class McpCallToolResp {
    result: string
    error: string

    constructor() {
        this.result = ""
        this.error = ""
    }

    static parse(json: string): McpCallToolResp {
        let resp = new McpCallToolResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("result")) resp.result = obj.get("result").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class McpListToolsResp {
    tools: string
    error: string

    constructor() {
        this.tools = ""
        this.error = ""
    }

    static parse(json: string): McpListToolsResp {
        let resp = new McpListToolsResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("tools")) resp.tools = obj.get("tools").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace mcp {
    export function callTool(msg: McpCallToolMsg, callback: string): void {
        _invokeAsync("mcp.callTool", msg.toJSON(), callback)
    }

    export function listTools(msg: McpListToolsMsg, callback: string): void {
        _invokeAsync("mcp.listTools", msg.toJSON(), callback)
    }

}
