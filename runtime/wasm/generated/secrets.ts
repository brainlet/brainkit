// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: secrets

export namespace secrets {
}

// Events
export class SecretsSetMsg {
    name: string
    value: string

    constructor(name: string, value: string) {
        this.name = name
        this.value = value
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("value", this.value)
        return obj.toString()
    }
}

export class SecretsGetMsg {
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

export class SecretsDeleteMsg {
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

export class SecretsListMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class SecretsRotateMsg {
    name: string
    newValue: string
    restart: bool

    constructor(name: string, newValue: string, restart: bool) {
        this.name = name
        this.newValue = newValue
        this.restart = restart
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("newValue", this.newValue)
        obj.setBool("restart", this.restart)
        return obj.toString()
    }
}

export class SecretsAccessedEvent {
    name: string
    accessor: string
    timestamp: string

    constructor(name: string, accessor: string, timestamp: string) {
        this.name = name
        this.accessor = accessor
        this.timestamp = timestamp
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("accessor", this.accessor)
        obj.setString("timestamp", this.timestamp)
        return obj.toString()
    }
}

export class SecretsStoredEvent {
    name: string
    version: i32
    timestamp: string

    constructor(name: string, version: i32, timestamp: string) {
        this.name = name
        this.version = version
        this.timestamp = timestamp
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setInt("version", this.version)
        obj.setString("timestamp", this.timestamp)
        return obj.toString()
    }
}

export class SecretsRotatedEvent {
    name: string
    version: i32
    restartedPlugins: string
    timestamp: string

    constructor(name: string, version: i32, restartedPlugins: string, timestamp: string) {
        this.name = name
        this.version = version
        this.restartedPlugins = restartedPlugins
        this.timestamp = timestamp
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setInt("version", this.version)
        if (this.restartedPlugins.length > 0) obj.set("restartedPlugins", JSONValue.parse(this.restartedPlugins))
        obj.setString("timestamp", this.timestamp)
        return obj.toString()
    }
}

export class SecretsDeletedEvent {
    name: string
    timestamp: string

    constructor(name: string, timestamp: string) {
        this.name = name
        this.timestamp = timestamp
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("timestamp", this.timestamp)
        return obj.toString()
    }
}

