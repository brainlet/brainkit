// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: peers

export namespace peers {
}

// Events
export class PeersListMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class PeersResolveMsg {
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

