package messages

import "encoding/json"

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
	ResultMeta
	ThreadID string `json:"threadId"`
}

func (MemoryCreateThreadResp) BusTopic() string { return "memory.createThread.result" }

type MemoryGetThreadResp struct {
	ResultMeta
	Thread json.RawMessage `json:"thread"`
}

func (MemoryGetThreadResp) BusTopic() string { return "memory.getThread.result" }

type MemoryListThreadsResp struct {
	ResultMeta
	Threads json.RawMessage `json:"threads"`
}

func (MemoryListThreadsResp) BusTopic() string { return "memory.listThreads.result" }

type MemorySaveResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

func (MemorySaveResp) BusTopic() string { return "memory.save.result" }

type MemoryRecallResp struct {
	ResultMeta
	Messages []MemoryMessage `json:"messages"`
}

func (MemoryRecallResp) BusTopic() string { return "memory.recall.result" }

type MemoryDeleteThreadResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

func (MemoryDeleteThreadResp) BusTopic() string { return "memory.deleteThread.result" }

// ── Shared types ──

type MemoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
