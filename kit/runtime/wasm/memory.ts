// runtime/wasm/memory.ts — Memory domain typed messages + namespace functions.

import { _invokeAsync } from "./host"

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
        obj.set("messages", JSONValue.parse(this.messagesJSON))
        return obj.toString()
    }
}

export namespace memory {
    export function recall(msg: MemoryRecallMsg, callback: string): void {
        _invokeAsync("memory.recall", msg.toJSON(), callback)
    }

    export function save(msg: MemorySaveMsg, callback: string): void {
        _invokeAsync("memory.save", msg.toJSON(), callback)
    }
}
