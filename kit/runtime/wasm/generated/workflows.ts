// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: workflows

export class WorkflowCancelMsg {
    runId: string

    constructor(runId: string) {
        this.runId = runId
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("runId", this.runId)
        return obj.toString()
    }
}

export class WorkflowResumeMsg {
    runId: string
    stepId: string
    data: string

    constructor(runId: string, stepId: string, data: string) {
        this.runId = runId
        this.stepId = stepId
        this.data = data
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("runId", this.runId)
        obj.setString("stepId", this.stepId)
        if (this.data.length > 0) obj.set("data", JSONValue.parse(this.data))
        return obj.toString()
    }
}

export class WorkflowRunMsg {
    name: string
    input: string

    constructor(name: string, input: string) {
        this.name = name
        this.input = input
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("name", this.name)
        if (this.input.length > 0) obj.set("input", JSONValue.parse(this.input))
        return obj.toString()
    }
}

export class WorkflowStatusMsg {
    runId: string

    constructor(runId: string) {
        this.runId = runId
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("runId", this.runId)
        return obj.toString()
    }
}

export class WorkflowCancelResp {
    ok: bool
    error: string

    constructor() {
        this.ok = false
        this.error = ""
    }

    static parse(json: string): WorkflowCancelResp {
        let resp = new WorkflowCancelResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("ok")) resp.ok = obj.getBool("ok")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WorkflowResumeResp {
    result: string
    error: string

    constructor() {
        this.result = ""
        this.error = ""
    }

    static parse(json: string): WorkflowResumeResp {
        let resp = new WorkflowResumeResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("result")) resp.result = obj.get("result").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WorkflowRunResp {
    result: string
    error: string

    constructor() {
        this.result = ""
        this.error = ""
    }

    static parse(json: string): WorkflowRunResp {
        let resp = new WorkflowRunResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("result")) resp.result = obj.get("result").toString()
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export class WorkflowStatusResp {
    status: string
    step: string
    error: string

    constructor() {
        this.status = ""
        this.step = ""
        this.error = ""
    }

    static parse(json: string): WorkflowStatusResp {
        let resp = new WorkflowStatusResp()
        let val = JSONValue.parse(json)
        if (val.isObject()) {
            let obj = val.asObject()
            if (obj.has("status")) resp.status = obj.getString("status")
            if (obj.has("step")) resp.step = obj.getString("step")
            if (obj.has("error")) resp.error = obj.getString("error")
        }
        return resp
    }
}

export namespace workflows {
    export function cancel(msg: WorkflowCancelMsg, callback: string): void {
        _invokeAsync("workflows.cancel", msg.toJSON(), callback)
    }

    export function resume(msg: WorkflowResumeMsg, callback: string): void {
        _invokeAsync("workflows.resume", msg.toJSON(), callback)
    }

    export function run(msg: WorkflowRunMsg, callback: string): void {
        _invokeAsync("workflows.run", msg.toJSON(), callback)
    }

    export function status(msg: WorkflowStatusMsg, callback: string): void {
        _invokeAsync("workflows.status", msg.toJSON(), callback)
    }

}
