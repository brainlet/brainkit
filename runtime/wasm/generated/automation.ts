// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: automation

export namespace automation {
}

// Events
export class AutomationDeployMsg {
    path: string
    manifest: string
    workflowSource: string
    adminSource: string

    constructor(path: string, manifest: string, workflowSource: string, adminSource: string) {
        this.path = path
        this.manifest = manifest
        this.workflowSource = workflowSource
        this.adminSource = adminSource
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        if (this.manifest.length > 0) obj.set("manifest", JSONValue.parse(this.manifest))
        obj.setString("workflowSource", this.workflowSource)
        obj.setString("adminSource", this.adminSource)
        return obj.toString()
    }
}

export class AutomationTeardownMsg {
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

export class AutomationListMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class AutomationInfoMsg {
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

