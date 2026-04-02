// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: workflow

export namespace workflow {
}

// Events
export class WorkflowRunMsg {
    workflowId: string
    input: string
    hostResults: string

    constructor(workflowId: string, input: string, hostResults: string) {
        this.workflowId = workflowId
        this.input = input
        this.hostResults = hostResults
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("workflowId", this.workflowId)
        if (this.input.length > 0) obj.set("input", JSONValue.parse(this.input))
        if (this.hostResults.length > 0) obj.set("hostResults", JSONValue.parse(this.hostResults))
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

export class WorkflowListMsg {
    workflowId: string

    constructor(workflowId: string) {
        this.workflowId = workflowId
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("workflowId", this.workflowId)
        return obj.toString()
    }
}

export class WorkflowHistoryMsg {
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

