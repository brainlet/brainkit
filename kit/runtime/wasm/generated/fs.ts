// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: fs

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

export class FsDeleteResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): FsDeleteResp {
        let resp = new FsDeleteResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class FsListResp {
    files: string
    error: string

    constructor() {
        this.files = ""
        this.error = ""
    }

    static parse(json: string): FsListResp {
        let resp = new FsListResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("files")) resp.files = obj.get("files").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class FsMkdirResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): FsMkdirResp {
        let resp = new FsMkdirResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class FsReadResp {
    data: string
    error: string

    constructor() {
        this.data = ""
        this.error = ""
    }

    static parse(json: string): FsReadResp {
        let resp = new FsReadResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("data")) resp.data = obj.getString("data")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class FsStatResp {
    size: i32
    isDir: bool
    modTime: string
    error: string

    constructor() {
        this.size = 0
        this.isDir = false
        this.modTime = ""
        this.error = ""
    }

    static parse(json: string): FsStatResp {
        let resp = new FsStatResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("size")) resp.size = obj.getInt("size")
            if (obj.has("isDir")) resp.isDir = obj.getBool("isDir")
            if (obj.has("modTime")) resp.modTime = obj.getString("modTime")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class FsWriteResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): FsWriteResp {
        let resp = new FsWriteResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace fs_ops {
    export function delete(msg: FsDeleteMsg, callback: string): void {
        _invokeAsync("fs.delete", msg.toJSON(), callback)
    }

    export function list(msg: FsListMsg, callback: string): void {
        _invokeAsync("fs.list", msg.toJSON(), callback)
    }

    export function mkdir(msg: FsMkdirMsg, callback: string): void {
        _invokeAsync("fs.mkdir", msg.toJSON(), callback)
    }

    export function read(msg: FsReadMsg, callback: string): void {
        _invokeAsync("fs.read", msg.toJSON(), callback)
    }

    export function stat(msg: FsStatMsg, callback: string): void {
        _invokeAsync("fs.stat", msg.toJSON(), callback)
    }

    export function write(msg: FsWriteMsg, callback: string): void {
        _invokeAsync("fs.write", msg.toJSON(), callback)
    }

}
