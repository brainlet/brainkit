// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: tools

export namespace tools {
}

// Events
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

