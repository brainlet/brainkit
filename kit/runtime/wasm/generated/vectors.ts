// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: vectors

export class VectorCreateIndexMsg {
    name: string
    dimension: i32
    metric: string

    constructor(name: string, dimension: i32, metric: string) {
        this.name = name
        this.dimension = dimension
        this.metric = metric
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setInt("dimension", this.dimension)
        obj.setString("metric", this.metric)
        return obj.toString()
    }
}

export class VectorDeleteIndexMsg {
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

export class VectorListIndexesMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class VectorQueryMsg {
    index: string
    embedding: string
    topK: i32
    filter: string

    constructor(index: string, embedding: string, topK: i32, filter: string) {
        this.index = index
        this.embedding = embedding
        this.topK = topK
        this.filter = filter
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("index", this.index)
        if (this.embedding.length > 0) obj.set("embedding", JSONValue.parse(this.embedding))
        obj.setInt("topK", this.topK)
        if (this.filter.length > 0) obj.set("filter", JSONValue.parse(this.filter))
        return obj.toString()
    }
}

export class VectorUpsertMsg {
    index: string
    vectors: string

    constructor(index: string, vectors: string) {
        this.index = index
        this.vectors = vectors
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("index", this.index)
        if (this.vectors.length > 0) obj.set("vectors", JSONValue.parse(this.vectors))
        return obj.toString()
    }
}

export class VectorCreateIndexResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): VectorCreateIndexResp {
        let resp = new VectorCreateIndexResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class VectorDeleteIndexResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): VectorDeleteIndexResp {
        let resp = new VectorDeleteIndexResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class VectorListIndexesResp {
    indexes: string
    error: string

    constructor() {
        this.indexes = ""
        this.error = ""
    }

    static parse(json: string): VectorListIndexesResp {
        let resp = new VectorListIndexesResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("indexes")) resp.indexes = obj.get("indexes").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class VectorQueryResp {
    matches: string
    error: string

    constructor() {
        this.matches = ""
        this.error = ""
    }

    static parse(json: string): VectorQueryResp {
        let resp = new VectorQueryResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("matches")) resp.matches = obj.get("matches").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class VectorUpsertResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): VectorUpsertResp {
        let resp = new VectorUpsertResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace vectors {
    export function createIndex(msg: VectorCreateIndexMsg, callback: string): void {
        _invokeAsync("vectors.createIndex", msg.toJSON(), callback)
    }

    export function deleteIndex(msg: VectorDeleteIndexMsg, callback: string): void {
        _invokeAsync("vectors.deleteIndex", msg.toJSON(), callback)
    }

    export function listIndexes(msg: VectorListIndexesMsg, callback: string): void {
        _invokeAsync("vectors.listIndexes", msg.toJSON(), callback)
    }

    export function query(msg: VectorQueryMsg, callback: string): void {
        _invokeAsync("vectors.query", msg.toJSON(), callback)
    }

    export function upsert(msg: VectorUpsertMsg, callback: string): void {
        _invokeAsync("vectors.upsert", msg.toJSON(), callback)
    }

}
