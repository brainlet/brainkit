// runtime/wasm/vectors.ts — Vectors domain typed messages + namespace functions.

import { _invokeAsync } from "./host"

export class VectorQueryMsg {
    index: string
    embeddingJSON: string
    topK: i32

    constructor(index: string, embeddingJSON: string, topK: i32) {
        this.index = index
        this.embeddingJSON = embeddingJSON
        this.topK = topK
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("index", this.index)
        obj.set("embedding", JSONValue.parse(this.embeddingJSON))
        obj.setInt("topK", this.topK)
        return obj.toString()
    }
}

export class VectorUpsertMsg {
    index: string
    vectorsJSON: string

    constructor(index: string, vectorsJSON: string) {
        this.index = index
        this.vectorsJSON = vectorsJSON
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("index", this.index)
        obj.set("vectors", JSONValue.parse(this.vectorsJSON))
        return obj.toString()
    }
}

export namespace vectors {
    export function query(msg: VectorQueryMsg, callback: string): void {
        _invokeAsync("vectors.query", msg.toJSON(), callback)
    }

    export function upsert(msg: VectorUpsertMsg, callback: string): void {
        _invokeAsync("vectors.upsert", msg.toJSON(), callback)
    }
}
