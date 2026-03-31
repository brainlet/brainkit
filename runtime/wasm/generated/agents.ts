// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: agents

export namespace agents {
}

// Events
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

