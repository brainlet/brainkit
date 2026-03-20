# Memory API Reference

Thread-based conversation memory. Create threads, save messages, and recall by query.

---

## Bus Topics

All memory operations use Ask (request/response) over the bus.

### memory.createThread

Create a new conversation thread.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"opts": {"title": string, "metadata": map[string]string}}` |
| **Response** | `{"threadId": string}` |

`opts` is optional. Both `title` and `metadata` within opts are optional.

### memory.getThread

Retrieve a thread by ID.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"threadId": string}` |
| **Response** | Thread object |

### memory.listThreads

List threads with optional filtering.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"filter": {"title": string, "limit": int}}` |
| **Response** | Thread array |

`filter` is optional. Both `title` and `limit` within filter are optional.

### memory.save

Save messages to a thread.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"threadId": string, "messages": MemoryMessage[]}` |
| **Response** | *(ack)* |

### memory.recall

Recall messages from a thread, optionally filtered by query.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"threadId": string, "query": string}` |
| **Response** | `{"messages": MemoryMessage[]}` |

### memory.deleteThread

Delete a thread by ID.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"threadId": string}` |
| **Response** | *(ack)* |

---

## SDK Messages

Typed messages for bus interactions. All implement `BusMessage`.

```go
import "github.com/brainlet/brainkit/sdk/messages"
```

### Request Messages

| Message | Fields | BusTopic() |
|---------|--------|------------|
| `MemoryCreateThreadMsg` | `Opts *MemoryCreateThreadOpts` | `"memory.createThread"` |
| `MemoryGetThreadMsg` | `ThreadID string` | `"memory.getThread"` |
| `MemoryListThreadsMsg` | `Filter *MemoryThreadFilter` | `"memory.listThreads"` |
| `MemorySaveMsg` | `ThreadID string`, `Messages []MemoryMessage` | `"memory.save"` |
| `MemoryRecallMsg` | `ThreadID string`, `Query string` | `"memory.recall"` |
| `MemoryDeleteThreadMsg` | `ThreadID string` | `"memory.deleteThread"` |

### Response Messages

| Message | Fields |
|---------|--------|
| `MemoryCreateThreadResp` | `ThreadID string` |
| `MemoryRecallResp` | `Messages []MemoryMessage` |

---

## Types

### MemoryCreateThreadOpts

```go
type MemoryCreateThreadOpts struct {
    Title    string            `json:"title,omitempty"`
    Metadata map[string]string `json:"metadata,omitempty"`
}
```

### MemoryThreadFilter

```go
type MemoryThreadFilter struct {
    Title string `json:"title,omitempty"`
    Limit int    `json:"limit,omitempty"`
}
```

### MemoryMessage

```go
type MemoryMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Role` | `string` | Message role (e.g. `"user"`, `"assistant"`, `"system"`) |
| `Content` | `string` | Message content |
