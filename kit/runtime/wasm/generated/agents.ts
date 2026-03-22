// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: agents

export class AgentDiscoverMsg {
    capability: string
    model: string
    status: string

    constructor(capability: string, model: string, status: string) {
        this.capability = capability
        this.model = model
        this.status = status
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("capability", this.capability)
        obj.setString("model", this.model)
        obj.setString("status", this.status)
        return obj.toString()
    }
}

export class AgentGetStatusMsg {
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

export class AgentListMsg {
    filter: string

    constructor(filter: string) {
        this.filter = filter
    }

    toJSON(): string {
        let obj = new JSONObject()
        if (this.filter.length > 0) obj.set("filter", JSONValue.parse(this.filter))
        return obj.toString()
    }
}

export class AgentMessageMsg {
    target: string
    payload: string

    constructor(target: string, payload: string) {
        this.target = target
        this.payload = payload
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("target", this.target)
        if (this.payload.length > 0) obj.set("payload", JSONValue.parse(this.payload))
        return obj.toString()
    }
}

export class AgentRequestMsg {
    name: string
    prompt: string

    constructor(name: string, prompt: string) {
        this.name = name
        this.prompt = prompt
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("prompt", this.prompt)
        return obj.toString()
    }
}

export class AgentSetStatusMsg {
    name: string
    status: string

    constructor(name: string, status: string) {
        this.name = name
        this.status = status
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("status", this.status)
        return obj.toString()
    }
}

export class AgentDiscoverResp {
    agents: string
    error: string

    constructor() {
        this.agents = ""
        this.error = ""
    }

    static parse(json: string): AgentDiscoverResp {
        let resp = new AgentDiscoverResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("agents")) resp.agents = obj.get("agents").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AgentGetStatusResp {
    name: string
    status: string
    error: string

    constructor() {
        this.name = ""
        this.status = ""
        this.error = ""
    }

    static parse(json: string): AgentGetStatusResp {
        let resp = new AgentGetStatusResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("name")) resp.name = obj.getString("name")
            if (obj.has("status")) resp.status = obj.getString("status")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AgentListResp {
    agents: string
    error: string

    constructor() {
        this.agents = ""
        this.error = ""
    }

    static parse(json: string): AgentListResp {
        let resp = new AgentListResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("agents")) resp.agents = obj.get("agents").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AgentMessageResp {
    delivered: bool
    error: string

    constructor() {
        this.delivered = false
        this.error = ""
    }

    static parse(json: string): AgentMessageResp {
        let resp = new AgentMessageResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("delivered")) resp.delivered = obj.getBool("delivered")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AgentRequestResp {
    text: string
    error: string

    constructor() {
        this.text = ""
        this.error = ""
    }

    static parse(json: string): AgentRequestResp {
        let resp = new AgentRequestResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("text")) resp.text = obj.getString("text")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AgentSetStatusResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): AgentSetStatusResp {
        let resp = new AgentSetStatusResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace agents {
    export function discover(msg: AgentDiscoverMsg, callback: string): void {
        _invokeAsync("agents.discover", msg.toJSON(), callback)
    }

    export function get-status(msg: AgentGetStatusMsg, callback: string): void {
        _invokeAsync("agents.get-status", msg.toJSON(), callback)
    }

    export function list(msg: AgentListMsg, callback: string): void {
        _invokeAsync("agents.list", msg.toJSON(), callback)
    }

    export function message(msg: AgentMessageMsg, callback: string): void {
        _invokeAsync("agents.message", msg.toJSON(), callback)
    }

    export function request(msg: AgentRequestMsg, callback: string): void {
        _invokeAsync("agents.request", msg.toJSON(), callback)
    }

    export function set-status(msg: AgentSetStatusMsg, callback: string): void {
        _invokeAsync("agents.set-status", msg.toJSON(), callback)
    }

}

// Events
export class AgentStreamMsg {
    name: string
    prompt: string
    streamTo: string

    constructor(name: string, prompt: string, streamTo: string) {
        this.name = name
        this.prompt = prompt
        this.streamTo = streamTo
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("prompt", this.prompt)
        obj.setString("streamTo", this.streamTo)
        return obj.toString()
    }
}

