// runtime/wasm/memory.ts — Memory domain typed messages + namespace functions.

import { _askAsync } from "./host"

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
    messagesJSON: string

    constructor(threadId: string, messagesJSON: string) {
        this.threadId = threadId
        this.messagesJSON = messagesJSON
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("threadId", this.threadId)
        obj.setRaw("messages", this.messagesJSON)
        return obj.toString()
    }
}

export namespace memory {
    export function recall(msg: MemoryRecallMsg, callback: string): void {
        _askAsync("memory.recall", msg.toJSON(), callback)
    }

    export function save(msg: MemorySaveMsg, callback: string): void {
        _askAsync("memory.save", msg.toJSON(), callback)
    }
}
