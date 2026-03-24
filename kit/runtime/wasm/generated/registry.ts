// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: registry

export namespace registry {
}

// Events
export class RegistryHasMsg {
    category: string
    name: string

    constructor(category: string, name: string) {
        this.category = category
        this.name = name
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("category", this.category)
        obj.setString("name", this.name)
        return obj.toString()
    }
}

export class RegistryListMsg {
    category: string

    constructor(category: string) {
        this.category = category
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("category", this.category)
        return obj.toString()
    }
}

export class RegistryResolveMsg {
    category: string
    name: string

    constructor(category: string, name: string) {
        this.category = category
        this.name = name
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("category", this.category)
        obj.setString("name", this.name)
        return obj.toString()
    }
}

