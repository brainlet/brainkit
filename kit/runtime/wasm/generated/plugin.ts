// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: plugin

export class PluginManifestMsg {
    owner: string
    name: string
    version: string
    description: string
    tools: string
    subscriptions: string
    events: string

    constructor(owner: string, name: string, version: string, description: string, tools: string, subscriptions: string, events: string) {
        this.owner = owner
        this.name = name
        this.version = version
        this.description = description
        this.tools = tools
        this.subscriptions = subscriptions
        this.events = events
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("owner", this.owner)
        obj.setString("name", this.name)
        obj.setString("version", this.version)
        obj.setString("description", this.description)
        if (this.tools.length > 0) obj.set("tools", JSONValue.parse(this.tools))
        if (this.subscriptions.length > 0) obj.set("subscriptions", JSONValue.parse(this.subscriptions))
        if (this.events.length > 0) obj.set("events", JSONValue.parse(this.events))
        return obj.toString()
    }
}

export class PluginStateGetMsg {
    key: string

    constructor(key: string) {
        this.key = key
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("key", this.key)
        return obj.toString()
    }
}

export class PluginStateSetMsg {
    key: string
    value: string

    constructor(key: string, value: string) {
        this.key = key
        this.value = value
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("key", this.key)
        obj.setString("value", this.value)
        return obj.toString()
    }
}

export class PluginManifestResp {
    registered: bool
    error: string

    constructor() {
        this.registered = false
        this.error = ""
    }

    static parse(json: string): PluginManifestResp {
        let resp = new PluginManifestResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("registered")) resp.registered = obj.getBool("registered")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class PluginStateGetResp {
    value: string
    error: string

    constructor() {
        this.value = ""
        this.error = ""
    }

    static parse(json: string): PluginStateGetResp {
        let resp = new PluginStateGetResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("value")) resp.value = obj.getString("value")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class PluginStateSetResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): PluginStateSetResp {
        let resp = new PluginStateSetResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace plugin {
    export function manifest(msg: PluginManifestMsg, callback: string): void {
        _invokeAsync("plugin.manifest", msg.toJSON(), callback)
    }

    export function stateGet(msg: PluginStateGetMsg, callback: string): void {
        _invokeAsync("plugin.state.get", msg.toJSON(), callback)
    }

    export function stateSet(msg: PluginStateSetMsg, callback: string): void {
        _invokeAsync("plugin.state.set", msg.toJSON(), callback)
    }

}

// Events
export class PluginRegisteredEvent {
    owner: string
    name: string
    version: string
    tools: i32

    constructor(owner: string, name: string, version: string, tools: i32) {
        this.owner = owner
        this.name = name
        this.version = version
        this.tools = tools
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("owner", this.owner)
        obj.setString("name", this.name)
        obj.setString("version", this.version)
        obj.setInt("tools", this.tools)
        return obj.toString()
    }
}

