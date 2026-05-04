package module

import (
	"context"
	"encoding/json"
)

// LifecycleDebugRegistry accepts scoped component debug snapshots for operator
// inspection. Snapshot functions must be read-only and must not control runtime
// behavior.
type LifecycleDebugRegistry interface {
	RegisterLifecycleDebug(context.Context, string, func() any) (Handle, error)
}

// LifecycleDebugSnapshot is the Brainkit-owned wire shape for runtime
// lifecycle diagnostics. It exposes counts and state, not implementation
// objects.
type LifecycleDebugSnapshot struct {
	Runtime    LifecycleRuntimeDebug     `json:"runtime"`
	Components []LifecycleDebugComponent `json:"components,omitempty"`
}

// LifecycleDebugComponent is a module-provided debug view.
type LifecycleDebugComponent struct {
	Name  string          `json:"name"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// LifecycleRuntimeDebug is the core runtime debug view.
type LifecycleRuntimeDebug struct {
	RuntimeID      string                  `json:"runtimeId,omitempty"`
	Namespace      string                  `json:"namespace,omitempty"`
	CallerID       string                  `json:"callerId,omitempty"`
	MountedModules int                     `json:"mountedModules"`
	ActiveHandlers int64                   `json:"activeHandlers"`
	Draining       bool                    `json:"draining"`
	Provider       LifecycleProviderDebug  `json:"provider"`
	Storage        LifecycleStorageDebug   `json:"storage"`
	Transport      LifecycleTransportDebug `json:"transport"`
}

// LifecycleProviderDebug reports provider-registry lifecycle counts.
type LifecycleProviderDebug struct {
	AIProviders      int   `json:"aiProviders"`
	VectorStores     int   `json:"vectorStores"`
	Storages         int   `json:"storages"`
	Closing          bool  `json:"closing"`
	Closed           bool  `json:"closed"`
	ActiveProbes     int64 `json:"activeProbes"`
	ActiveOperations int64 `json:"activeOperations"`
}

// LifecycleStorageDebug reports storage bridge ownership state.
type LifecycleStorageDebug struct {
	Closing      bool     `json:"closing"`
	BridgeCount  int      `json:"bridgeCount"`
	BridgeNames  []string `json:"bridgeNames,omitempty"`
	ActiveCloses int      `json:"activeCloses"`
}

// LifecycleTransportDebug reports transport/router lifecycle counts.
type LifecycleTransportDebug struct {
	Kind                   string               `json:"kind,omitempty"`
	OwnsTransport          bool                 `json:"ownsTransport"`
	ActiveSubscriptions    int64                `json:"activeSubscriptions"`
	ActiveStreamHeartbeats int                  `json:"activeStreamHeartbeats"`
	CallerClosed           bool                 `json:"callerClosed"`
	CallerPendingCalls     int                  `json:"callerPendingCalls"`
	CallerStreamDrains     int64                `json:"callerStreamDrains"`
	ClosingRouter          bool                 `json:"closingRouter"`
	ClosingCaller          bool                 `json:"closingCaller"`
	ClosingTransport       bool                 `json:"closingTransport"`
	ClosedRouter           bool                 `json:"closedRouter"`
	ClosedCaller           bool                 `json:"closedCaller"`
	ClosedTransport        bool                 `json:"closedTransport"`
	Router                 LifecycleRouterDebug `json:"router"`
}

// LifecycleRouterDebug reports command/router handler bookkeeping.
type LifecycleRouterDebug struct {
	Handlers        int            `json:"handlers"`
	StartedHandlers int            `json:"startedHandlers"`
	StoppedHandlers int            `json:"stoppedHandlers"`
	Topics          map[string]int `json:"topics,omitempty"`
}
