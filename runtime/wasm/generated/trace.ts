// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: trace

export namespace trace {
}

// Events
export class TraceGetMsg {
    traceId: string

    constructor(traceId: string) {
        this.traceId = traceId
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("traceId", this.traceId)
        return obj.toString()
    }
}

export class TraceListMsg {
    source: string
    status: string
    minDuration: i32
    limit: i32

    constructor(source: string, status: string, minDuration: i32, limit: i32) {
        this.source = source
        this.status = status
        this.minDuration = minDuration
        this.limit = limit
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        obj.setString("status", this.status)
        obj.setInt("minDuration", this.minDuration)
        obj.setInt("limit", this.limit)
        return obj.toString()
    }
}

