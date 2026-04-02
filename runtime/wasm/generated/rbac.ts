// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: rbac

export namespace rbac {
}

// Events
export class RBACAssignMsg {
    source: string
    role: string

    constructor(source: string, role: string) {
        this.source = source
        this.role = role
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        obj.setString("role", this.role)
        return obj.toString()
    }
}

export class RBACRevokeMsg {
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

export class RBACListMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class RBACRolesMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

