// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: plugin

export namespace plugin {
}

// Events
export class PluginManifestMsg {
    owner: string
    name: string
    version: string
    description: string
    tools: string
    subscriptions: string
    events: string
    host_functions: string

    constructor(owner: string, name: string, version: string, description: string, tools: string, subscriptions: string, events: string, host_functions: string) {
        this.owner = owner
        this.name = name
        this.version = version
        this.description = description
        this.tools = tools
        this.subscriptions = subscriptions
        this.events = events
        this.host_functions = host_functions
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
        if (this.host_functions.length > 0) obj.set("host_functions", JSONValue.parse(this.host_functions))
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

export class PluginStartMsg {
    name: string
    binary: string
    env: string
    config: string
    role: string

    constructor(name: string, binary: string, env: string, config: string, role: string) {
        this.name = name
        this.binary = binary
        this.env = env
        this.config = config
        this.role = role
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("binary", this.binary)
        if (this.env.length > 0) obj.set("env", JSONValue.parse(this.env))
        if (this.config.length > 0) obj.set("config", JSONValue.parse(this.config))
        obj.setString("role", this.role)
        return obj.toString()
    }
}

export class PluginStopMsg {
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

export class PluginRestartMsg {
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

export class PluginListRunningMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class PluginStatusMsg {
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

export class PluginStartedEvent {
    name: string
    pid: i32
    version: string

    constructor(name: string, pid: i32, version: string) {
        this.name = name
        this.pid = pid
        this.version = version
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setInt("pid", this.pid)
        obj.setString("version", this.version)
        return obj.toString()
    }
}

export class PluginStoppedEvent {
    name: string
    reason: string

    constructor(name: string, reason: string) {
        this.name = name
        this.reason = reason
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("reason", this.reason)
        return obj.toString()
    }
}

