// runtime/wasm/fs.ts — Filesystem domain typed messages + namespace functions.

import { _askAsync } from "./host"

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

    constructor(path: string, pattern: string = "") {
        this.path = path
        this.pattern = pattern
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        if (this.pattern.length > 0) {
            obj.setString("pattern", this.pattern)
        }
        return obj.toString()
    }
}

export namespace fs_ops {
    export function read(msg: FsReadMsg, callback: string): void {
        _askAsync("fs.read", msg.toJSON(), callback)
    }

    export function write(msg: FsWriteMsg, callback: string): void {
        _askAsync("fs.write", msg.toJSON(), callback)
    }

    export function list(msg: FsListMsg, callback: string): void {
        _askAsync("fs.list", msg.toJSON(), callback)
    }
}
