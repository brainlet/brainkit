// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: test

export namespace testing {
}

// Events
export class TestRunMsg {
    dir: string
    pattern: string
    skipAI: bool

    constructor(dir: string, pattern: string, skipAI: bool) {
        this.dir = dir
        this.pattern = pattern
        this.skipAI = skipAI
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("dir", this.dir)
        obj.setString("pattern", this.pattern)
        obj.setBool("skipAI", this.skipAI)
        return obj.toString()
    }
}

