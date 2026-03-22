// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: stream

export namespace stream {
}

// Events
export class StreamChunk {
    streamId: string
    seq: i32
    delta: string
    done: bool
    final: string

    constructor(streamId: string, seq: i32, delta: string, done: bool, final: string) {
        this.streamId = streamId
        this.seq = seq
        this.delta = delta
        this.done = done
        this.final = final
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("streamId", this.streamId)
        obj.setInt("seq", this.seq)
        obj.setString("delta", this.delta)
        obj.setBool("done", this.done)
        if (this.final.length > 0) obj.set("final", JSONValue.parse(this.final))
        return obj.toString()
    }
}

