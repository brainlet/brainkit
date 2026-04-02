// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: gateway

export namespace gateway {
}

// Events
export class GatewayRouteAddMsg {
    method: string
    path: string
    topic: string
    type: string
    owner: string

    constructor(method: string, path: string, topic: string, type: string, owner: string) {
        this.method = method
        this.path = path
        this.topic = topic
        this.type = type
        this.owner = owner
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("method", this.method)
        obj.setString("path", this.path)
        obj.setString("topic", this.topic)
        obj.setString("type", this.type)
        obj.setString("owner", this.owner)
        return obj.toString()
    }
}

export class GatewayRouteRemoveMsg {
    method: string
    path: string
    owner: string

    constructor(method: string, path: string, owner: string) {
        this.method = method
        this.path = path
        this.owner = owner
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("method", this.method)
        obj.setString("path", this.path)
        obj.setString("owner", this.owner)
        return obj.toString()
    }
}

export class GatewayRouteListMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class GatewayStatusMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

