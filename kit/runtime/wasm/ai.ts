// runtime/wasm/ai.ts — AI domain typed messages + namespace functions.

import { _invokeAsync } from "./host"

// ── Typed Messages ──

export class AiGenerateMsg {
    model: string
    prompt: string

    constructor(model: string, prompt: string) {
        this.model = model
        this.prompt = prompt
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("model", this.model)
        obj.setString("prompt", this.prompt)
        return obj.toString()
    }
}

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

// ── Typed Responses ──

export class AiGenerateResp {
    text: string
    promptTokens: i32
    completionTokens: i32

    constructor() {
        this.text = ""
        this.promptTokens = 0
        this.completionTokens = 0
    }

    static parse(json: string): AiGenerateResp {
        let resp = new AiGenerateResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.toObject()
            resp.text = obj.getString("text")
            let usage = obj.getObject("usage")
            if (usage != null) {
                resp.promptTokens = usage.getInteger("promptTokens") as i32
                resp.completionTokens = usage.getInteger("completionTokens") as i32
            }
        }
        return resp
    }
}

export class AiEmbedResp {
    embedding: string // JSON array as string — parse externally

    constructor() {
        this.embedding = "[]"
    }

    static parse(json: string): AiEmbedResp {
        let resp = new AiEmbedResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let embVal = val.toObject().get("embedding")
            if (embVal != null) {
                resp.embedding = embVal.toString()
            }
        }
        return resp
    }
}

// ── Namespace Functions ──

export namespace ai {
    export function generate(msg: AiGenerateMsg, callback: string): void {
        _invokeAsync("ai.generate", msg.toJSON(), callback)
    }

    export function embed(msg: AiEmbedMsg, callback: string): void {
        _invokeAsync("ai.embed", msg.toJSON(), callback)
    }
}
