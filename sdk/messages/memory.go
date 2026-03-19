package messages

// ── Options/Filters ──

type MemoryCreateThreadOpts struct {
	Title    string            `json:"title,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type MemoryThreadFilter struct {
	Title string `json:"title,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

// ── Requests ──

type MemoryCreateThreadMsg struct {
	Opts *MemoryCreateThreadOpts `json:"opts,omitempty"`
}

func (MemoryCreateThreadMsg) BusTopic() string { return "memory.createThread" }

type MemoryGetThreadMsg struct {
	ThreadID string `json:"threadId"`
}

func (MemoryGetThreadMsg) BusTopic() string { return "memory.getThread" }

type MemoryListThreadsMsg struct {
	Filter *MemoryThreadFilter `json:"filter,omitempty"`
}

func (MemoryListThreadsMsg) BusTopic() string { return "memory.listThreads" }

type MemorySaveMsg struct {
	ThreadID string          `json:"threadId"`
	Messages []MemoryMessage `json:"messages"`
}

func (MemorySaveMsg) BusTopic() string { return "memory.save" }

type MemoryRecallMsg struct {
	ThreadID string `json:"threadId"`
	Query    string `json:"query"`
}

func (MemoryRecallMsg) BusTopic() string { return "memory.recall" }

type MemoryDeleteThreadMsg struct {
	ThreadID string `json:"threadId"`
}

func (MemoryDeleteThreadMsg) BusTopic() string { return "memory.deleteThread" }

// ── Responses ──

type MemoryCreateThreadResp struct {
	ThreadID string `json:"threadId"`
}

type MemoryRecallResp struct {
	Messages []MemoryMessage `json:"messages"`
}

// ── Shared types ──

type MemoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
