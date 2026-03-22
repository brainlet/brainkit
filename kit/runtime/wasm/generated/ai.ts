// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: ai

export class AiEmbedMsg {
    model: string
    value: string

    constructor(model: string, value: string) {
        this.model = model
        this.value = value
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("model", this.model)
        obj.setString("value", this.value)
        return obj.toString()
    }
}

export class AiEmbedManyMsg {
    model: string
    values: string

    constructor(model: string, values: string) {
        this.model = model
        this.values = values
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("model", this.model)
        if (this.values.length > 0) obj.set("values", JSONValue.parse(this.values))
        return obj.toString()
    }
}

export class AiGenerateMsg {
    model: string
    prompt: string
    messages: string
    tools: string
    schema: string

    constructor(model: string, prompt: string, messages: string, tools: string, schema: string) {
        this.model = model
        this.prompt = prompt
        this.messages = messages
        this.tools = tools
        this.schema = schema
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("model", this.model)
        obj.setString("prompt", this.prompt)
        if (this.messages.length > 0) obj.set("messages", JSONValue.parse(this.messages))
        if (this.tools.length > 0) obj.set("tools", JSONValue.parse(this.tools))
        if (this.schema.length > 0) obj.set("schema", JSONValue.parse(this.schema))
        return obj.toString()
    }
}

export class AiGenerateObjectMsg {
    model: string
    prompt: string
    schema: string

    constructor(model: string, prompt: string, schema: string) {
        this.model = model
        this.prompt = prompt
        this.schema = schema
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("model", this.model)
        obj.setString("prompt", this.prompt)
        if (this.schema.length > 0) obj.set("schema", JSONValue.parse(this.schema))
        return obj.toString()
    }
}

export class AiEmbedResp {
    embedding: string
    error: string

    constructor() {
        this.embedding = ""
        this.error = ""
    }

    static parse(json: string): AiEmbedResp {
        let resp = new AiEmbedResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("embedding")) resp.embedding = obj.get("embedding").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AiEmbedManyResp {
    embeddings: string
    error: string

    constructor() {
        this.embeddings = ""
        this.error = ""
    }

    static parse(json: string): AiEmbedManyResp {
        let resp = new AiEmbedManyResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("embeddings")) resp.embeddings = obj.get("embeddings").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AiGenerateResp {
    text: string
    toolCalls: string
    usage: string
    error: string

    constructor() {
        this.text = ""
        this.toolCalls = ""
        this.usage = ""
        this.error = ""
    }

    static parse(json: string): AiGenerateResp {
        let resp = new AiGenerateResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("text")) resp.text = obj.getString("text")
            if (obj.has("toolCalls")) resp.toolCalls = obj.get("toolCalls").toString()
            if (obj.has("usage")) resp.usage = obj.get("usage").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class AiGenerateObjectResp {
    object: string
    error: string

    constructor() {
        this.object = ""
        this.error = ""
    }

    static parse(json: string): AiGenerateObjectResp {
        let resp = new AiGenerateObjectResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("object")) resp.object = obj.get("object").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace ai {
    export function embed(msg: AiEmbedMsg, callback: string): void {
        _invokeAsync("ai.embed", msg.toJSON(), callback)
    }

    export function embedMany(msg: AiEmbedManyMsg, callback: string): void {
        _invokeAsync("ai.embedMany", msg.toJSON(), callback)
    }

    export function generate(msg: AiGenerateMsg, callback: string): void {
        _invokeAsync("ai.generate", msg.toJSON(), callback)
    }

    export function generateObject(msg: AiGenerateObjectMsg, callback: string): void {
        _invokeAsync("ai.generateObject", msg.toJSON(), callback)
    }

}

// Events
export class AiStreamMsg {
    model: string
    prompt: string
    messages: string
    streamTo: string

    constructor(model: string, prompt: string, messages: string, streamTo: string) {
        this.model = model
        this.prompt = prompt
        this.messages = messages
        this.streamTo = streamTo
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("model", this.model)
        obj.setString("prompt", this.prompt)
        if (this.messages.length > 0) obj.set("messages", JSONValue.parse(this.messages))
        obj.setString("streamTo", this.streamTo)
        return obj.toString()
    }
}

