// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: fs

export namespace fs_ops {
}

// Events
export class FsReadMsg {
    path: string

    constructor(path: string) {
        this.path = path
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        return obj.toString()
    }
}

export class FsWriteMsg {
    path: string
    data: string

    constructor(path: string, data: string) {
        this.path = path
        this.data = data
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        obj.setString("data", this.data)
        return obj.toString()
    }
}

export class FsListMsg {
    path: string
    pattern: string

    constructor(path: string, pattern: string) {
        this.path = path
        this.pattern = pattern
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        obj.setString("pattern", this.pattern)
        return obj.toString()
    }
}

export class FsStatMsg {
    path: string

    constructor(path: string) {
        this.path = path
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        return obj.toString()
    }
}

export class FsDeleteMsg {
    path: string

    constructor(path: string) {
        this.path = path
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        return obj.toString()
    }
}

export class FsMkdirMsg {
    path: string

    constructor(path: string) {
        this.path = path
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        return obj.toString()
    }
}

