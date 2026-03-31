// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: kit

export namespace kit {
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

