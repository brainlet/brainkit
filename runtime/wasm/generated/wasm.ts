// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: wasm

export namespace wasm_ops {
}

// Events
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

export class WasmListMsg {

    toJSON(): string {
        let obj = new JSONObject()
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

export class WasmAllowlistGetMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class WasmAllowlistSetMsg {
    allowlist: string

    constructor(allowlist: string) {
        this.allowlist = allowlist
    }

    toJSON(): string {
        let obj = new JSONObject()
        if (this.allowlist.length > 0) obj.set("allowlist", JSONValue.parse(this.allowlist))
        return obj.toString()
    }
}

export class WasmAllowlistAddMsg {
    command: string

    constructor(command: string) {
        this.command = command
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("command", this.command)
        return obj.toString()
    }
}

export class WasmAllowlistRemoveMsg {
    command: string

    constructor(command: string) {
        this.command = command
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("command", this.command)
        return obj.toString()
    }
}

