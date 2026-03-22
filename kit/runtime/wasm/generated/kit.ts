// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: kit

export class KitDeployMsg {
    source: string
    code: string

    constructor(source: string, code: string) {
        this.source = source
        this.code = code
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        obj.setString("code", this.code)
        return obj.toString()
    }
}

export class KitListMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class KitRedeployMsg {
    source: string
    code: string

    constructor(source: string, code: string) {
        this.source = source
        this.code = code
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        obj.setString("code", this.code)
        return obj.toString()
    }
}

export class KitTeardownMsg {
    source: string

    constructor(source: string) {
        this.source = source
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        return obj.toString()
    }
}

export class KitDeployResp {
    deployed: bool
    resources: string
    error: string

    constructor() {
        this.deployed = false
        this.resources = ""
        this.error = ""
    }

    static parse(json: string): KitDeployResp {
        let resp = new KitDeployResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("deployed")) resp.deployed = obj.getBool("deployed")
            if (obj.has("resources")) resp.resources = obj.get("resources").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class KitListResp {
    deployments: string
    error: string

    constructor() {
        this.deployments = ""
        this.error = ""
    }

    static parse(json: string): KitListResp {
        let resp = new KitListResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("deployments")) resp.deployments = obj.get("deployments").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class KitRedeployResp {
    deployed: bool
    resources: string
    error: string

    constructor() {
        this.deployed = false
        this.resources = ""
        this.error = ""
    }

    static parse(json: string): KitRedeployResp {
        let resp = new KitRedeployResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("deployed")) resp.deployed = obj.getBool("deployed")
            if (obj.has("resources")) resp.resources = obj.get("resources").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class KitTeardownResp {
    removed: i32
    error: string

    constructor() {
        this.removed = 0
        this.error = ""
    }

    static parse(json: string): KitTeardownResp {
        let resp = new KitTeardownResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("removed")) resp.removed = obj.getInt("removed")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace kit {
    export function deploy(msg: KitDeployMsg, callback: string): void {
        _invokeAsync("kit.deploy", msg.toJSON(), callback)
    }

    export function list(msg: KitListMsg, callback: string): void {
        _invokeAsync("kit.list", msg.toJSON(), callback)
    }

    export function redeploy(msg: KitRedeployMsg, callback: string): void {
        _invokeAsync("kit.redeploy", msg.toJSON(), callback)
    }

    export function teardown(msg: KitTeardownMsg, callback: string): void {
        _invokeAsync("kit.teardown", msg.toJSON(), callback)
    }

}

// Events
export class KitDeployedEvent {
    source: string
    resources: string

    constructor(source: string, resources: string) {
        this.source = source
        this.resources = resources
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        if (this.resources.length > 0) obj.set("resources", JSONValue.parse(this.resources))
        return obj.toString()
    }
}

export class KitTeardownedEvent {
    source: string
    removed: i32

    constructor(source: string, removed: i32) {
        this.source = source
        this.removed = removed
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        obj.setInt("removed", this.removed)
        return obj.toString()
    }
}

