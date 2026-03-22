// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: memory

export class MemoryCreateThreadMsg {
    opts: string

    constructor(opts: string) {
        this.opts = opts
    }

    toJSON(): string {
        let obj = new JSONObject()
        if (this.opts.length > 0) obj.set("opts", JSONValue.parse(this.opts))
        return obj.toString()
    }
}

export class MemoryDeleteThreadMsg {
    threadId: string

    constructor(threadId: string) {
        this.threadId = threadId
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("threadId", this.threadId)
        return obj.toString()
    }
}

export class MemoryGetThreadMsg {
    threadId: string

    constructor(threadId: string) {
        this.threadId = threadId
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("threadId", this.threadId)
        return obj.toString()
    }
}

export class MemoryListThreadsMsg {
    filter: string

    constructor(filter: string) {
        this.filter = filter
    }

    toJSON(): string {
        let obj = new JSONObject()
        if (this.filter.length > 0) obj.set("filter", JSONValue.parse(this.filter))
        return obj.toString()
    }
}

export class MemoryRecallMsg {
    threadId: string
    query: string

    constructor(threadId: string, query: string) {
        this.threadId = threadId
        this.query = query
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("threadId", this.threadId)
        obj.setString("query", this.query)
        return obj.toString()
    }
}

export class MemorySaveMsg {
    threadId: string
    messages: string

    constructor(threadId: string, messages: string) {
        this.threadId = threadId
        this.messages = messages
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("threadId", this.threadId)
        if (this.messages.length > 0) obj.set("messages", JSONValue.parse(this.messages))
        return obj.toString()
    }
}

export class MemoryCreateThreadResp {
    threadId: string
    error: string

    constructor() {
        this.threadId = ""
        this.error = ""
    }

    static parse(json: string): MemoryCreateThreadResp {
        let resp = new MemoryCreateThreadResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("threadId")) resp.threadId = obj.getString("threadId")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class MemoryDeleteThreadResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): MemoryDeleteThreadResp {
        let resp = new MemoryDeleteThreadResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class MemoryGetThreadResp {
    thread: string
    error: string

    constructor() {
        this.thread = ""
        this.error = ""
    }

    static parse(json: string): MemoryGetThreadResp {
        let resp = new MemoryGetThreadResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("thread")) resp.thread = obj.get("thread").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class MemoryListThreadsResp {
    threads: string
    error: string

    constructor() {
        this.threads = ""
        this.error = ""
    }

    static parse(json: string): MemoryListThreadsResp {
        let resp = new MemoryListThreadsResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("threads")) resp.threads = obj.get("threads").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class MemoryRecallResp {
    messages: string
    error: string

    constructor() {
        this.messages = ""
        this.error = ""
    }

    static parse(json: string): MemoryRecallResp {
        let resp = new MemoryRecallResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("messages")) resp.messages = obj.get("messages").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class MemorySaveResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): MemorySaveResp {
        let resp = new MemorySaveResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace memory {
    export function createThread(msg: MemoryCreateThreadMsg, callback: string): void {
        _invokeAsync("memory.createThread", msg.toJSON(), callback)
    }

    export function deleteThread(msg: MemoryDeleteThreadMsg, callback: string): void {
        _invokeAsync("memory.deleteThread", msg.toJSON(), callback)
    }

    export function getThread(msg: MemoryGetThreadMsg, callback: string): void {
        _invokeAsync("memory.getThread", msg.toJSON(), callback)
    }

    export function listThreads(msg: MemoryListThreadsMsg, callback: string): void {
        _invokeAsync("memory.listThreads", msg.toJSON(), callback)
    }

    export function recall(msg: MemoryRecallMsg, callback: string): void {
        _invokeAsync("memory.recall", msg.toJSON(), callback)
    }

    export function save(msg: MemorySaveMsg, callback: string): void {
        _invokeAsync("memory.save", msg.toJSON(), callback)
    }

}
