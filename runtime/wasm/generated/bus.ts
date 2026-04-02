// AUTO-GENERATED from sdk/messages — do not edit.
// Domain: bus

export namespace bus {
}

// Events
export class HandlerFailedEvent {
    topic: string
    source: string
    error: string
    retryCount: i32
    willRetry: bool

    constructor(topic: string, source: string, error: string, retryCount: i32, willRetry: bool) {
        this.topic = topic
        this.source = source
        this.error = error
        this.retryCount = retryCount
        this.willRetry = willRetry
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("topic", this.topic)
        obj.setString("source", this.source)
        obj.setString("error", this.error)
        obj.setInt("retryCount", this.retryCount)
        obj.setBool("willRetry", this.willRetry)
        return obj.toString()
    }
}

export class HandlerExhaustedEvent {
    topic: string
    source: string
    error: string
    retryCount: i32

    constructor(topic: string, source: string, error: string, retryCount: i32) {
        this.topic = topic
        this.source = source
        this.error = error
        this.retryCount = retryCount
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("topic", this.topic)
        obj.setString("source", this.source)
        obj.setString("error", this.error)
        obj.setInt("retryCount", this.retryCount)
        return obj.toString()
    }
}

export class PermissionDeniedEvent {
    source: string
    topic: string
    action: string
    role: string
    reason: string

    constructor(source: string, topic: string, action: string, role: string, reason: string) {
        this.source = source
        this.topic = topic
        this.action = action
        this.role = role
        this.reason = reason
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        obj.setString("topic", this.topic)
        obj.setString("action", this.action)
        obj.setString("role", this.role)
        obj.setString("reason", this.reason)
        return obj.toString()
    }
}

export class ReplyDeniedEvent {
    source: string
    topic: string
    correlationId: string
    reason: string

    constructor(source: string, topic: string, correlationId: string, reason: string) {
        this.source = source
        this.topic = topic
        this.correlationId = correlationId
        this.reason = reason
    }

    toJSON(): string {
        let obj = new JSONObject()
        obj.setString("source", this.source)
        obj.setString("topic", this.topic)
        obj.setString("correlationId", this.correlationId)
        obj.setString("reason", this.reason)
        return obj.toString()
    }
}

