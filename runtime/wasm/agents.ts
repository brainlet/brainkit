// runtime/wasm/agents.ts — Agents domain typed messages + namespace functions.

import { _askAsync, _send } from "./host"

export class AgentRequestMsg {
    name: string
    prompt: string

    constructor(name: string, prompt: string) {
        this.name = name
        this.prompt = prompt
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        obj.setString("prompt", this.prompt)
        return obj.toString()
    }
}

export class AgentRequestResp {
    text: string

    constructor() { this.text = "" }

    static parse(json: string): AgentRequestResp {
        let resp = new AgentRequestResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            resp.text = val.toObject().getString("text")
        }
        return resp
    }
}

export class AgentMessageMsg {
    target: string
    payload: string // JSON

    constructor(target: string, payload: string) {
        this.target = target
        this.payload = payload
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("target", this.target)
        obj.setRaw("payload", this.payload)
        return obj.toString()
    }
}

export namespace agents {
    export function request(msg: AgentRequestMsg, callback: string): void {
        _askAsync("agents.request", msg.toJSON(), callback)
    }

    export function message(msg: AgentMessageMsg): void {
        _send("agents.message", msg.toJSON())
    }
}
