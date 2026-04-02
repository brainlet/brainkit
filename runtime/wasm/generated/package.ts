// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: package

export namespace pkg {
}

// Events
export class PackageDeployMsg {
    path: string
    manifest: string
    files: string

    constructor(path: string, manifest: string, files: string) {
        this.path = path
        this.manifest = manifest
        this.files = files
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        if (this.manifest.length > 0) obj.set("manifest", JSONValue.parse(this.manifest))
        if (this.files.length > 0) obj.set("files", JSONValue.parse(this.files))
        return obj.toString()
    }
}

export class PackageTeardownMsg {
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

export class PackageRedeployMsg {
    path: string

    constructor(path: string) {
        this.path = path
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("path", this.path)
        return obj.toString()
    }
}

export class PackageListDeployedMsg {

    toJSON(): string {
        let obj = new JSONObject()
        return obj.toString()
    }
}

export class PackageDeployInfoMsg {
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

