// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: mcp

export namespace mcp {
}

// Events
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

