// runtime/wasm/vectors.ts — Vectors domain typed messages + namespace functions.

import { _askAsync } from "./host"

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
        obj.setRaw("embedding", this.embeddingJSON)
        obj.setInteger("topK", this.topK as i64)
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
        obj.setRaw("vectors", this.vectorsJSON)
        return obj.toString()
    }
}

export namespace vectors {
    export function query(msg: VectorQueryMsg, callback: string): void {
        _askAsync("vectors.query", msg.toJSON(), callback)
    }

    export function upsert(msg: VectorUpsertMsg, callback: string): void {
        _askAsync("vectors.upsert", msg.toJSON(), callback)
    }
}
