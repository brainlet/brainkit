// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: packages

export namespace packages {
}

// Events
export class PackagesSearchMsg {
    query: string
    capabilities: string

    constructor(query: string, capabilities: string) {
        this.query = query
        this.capabilities = capabilities
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("query", this.query)
        if (this.capabilities.length > 0) obj.set("capabilities", JSONValue.parse(this.capabilities))
        return obj.toString()
    }
}

export class PackagesInstallMsg {
    name: string
    version: string

    constructor(name: string, version: string) {
        this.name = name
        this.version = version
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("version", this.version)
        return obj.toString()
    }
}

export class PackagesRemoveMsg {
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

export class PackagesUpdateMsg {
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

export class PackagesListMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class PackagesInfoMsg {
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

