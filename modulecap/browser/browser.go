// Package browsercap defines module-facing browser lifecycle contracts.
package browsercap

import (
	"context"
	"time"
)

// Scope describes how a browser resource is shared across Mastra threads.
type Scope string

const (
	ScopeThread Scope = "thread"
	ScopeShared Scope = "shared"
)

// LaunchRequest asks the browser manager to launch a local CDP-capable browser.
type LaunchRequest struct {
	Provider       string   `json:"provider,omitempty"`
	ThreadID       string   `json:"threadId,omitempty"`
	Scope          Scope    `json:"scope,omitempty"`
	ExecutablePath string   `json:"executablePath,omitempty"`
	ProfileDir     string   `json:"profileDir,omitempty"`
	Headless       *bool    `json:"headless,omitempty"`
	Args           []string `json:"args,omitempty"`
}

// SessionInfo is the stable lifecycle view of a browser resource.
type SessionInfo struct {
	ID                   string    `json:"id"`
	Provider             string    `json:"provider,omitempty"`
	ThreadID             string    `json:"threadId,omitempty"`
	Scope                Scope     `json:"scope,omitempty"`
	CDPURL               string    `json:"cdpUrl,omitempty"`
	WebSocketDebuggerURL string    `json:"webSocketDebuggerUrl,omitempty"`
	ProfileDir           string    `json:"profileDir,omitempty"`
	ExecutablePath       string    `json:"executablePath,omitempty"`
	PID                  int       `json:"pid,omitempty"`
	External             bool      `json:"external,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
}

// DebugSnapshot reports lifecycle counters for browser ownership.
type DebugSnapshot struct {
	Closed           bool `json:"closed"`
	Closing          bool `json:"closing"`
	Sessions         int  `json:"sessions"`
	OwnedProcesses   int  `json:"ownedProcesses"`
	OwnedProfiles    int  `json:"ownedProfiles"`
	ExternalSessions int  `json:"externalSessions"`
}

// Manager owns browser/CDP lifecycle for browser provider modules.
type Manager interface {
	Launch(context.Context, LaunchRequest) (SessionInfo, error)
	Close(context.Context, string) error
	List() []SessionInfo
	DebugSnapshot() DebugSnapshot
}
