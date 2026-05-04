package jsruntime

import (
	runtimecap "github.com/brainlet/brainkit/modulecap/runtime"
)

// JSRuntimeDebugSnapshot reports JS runtime lifecycle counters for operator
// inspect/debug surfaces.
func (r *Runtime) JSRuntimeDebugSnapshot() runtimecap.DebugSnapshot {
	if r == nil {
		return runtimecap.DebugSnapshot{Phase: "absent", Closed: true}
	}
	snapshot := runtimecap.DebugSnapshot{Phase: "active"}

	r.closeMu.Lock()
	closing := r.closing
	closed := r.closed
	closeMode := r.closeMode
	r.closeMu.Unlock()
	snapshot.BridgeClosing = closing
	snapshot.Closed = closed
	if closed {
		snapshot.Phase = runtimeClosePhase(closeMode)
	} else if closing {
		snapshot.Phase = "closing"
	}
	if r.bridge != nil {
		bridge := r.bridge.DebugSnapshot()
		snapshot.BridgeClosing = bridge.Closing
		snapshot.BridgeClosed = bridge.Closed
		snapshot.BridgeGoroutines = bridge.ActiveGoroutines
	}

	if r.deploymentMgr != nil {
		deployments := r.deploymentMgr.ListDeployments()
		snapshot.ActiveDeployments = len(deployments)
		resources, _ := r.deploymentMgr.ListResources()
		snapshot.ResourceCount = len(resources)
		if len(resources) > 0 {
			snapshot.ResourcesByType = make(map[string]int)
			for _, resource := range resources {
				snapshot.ResourcesByType[resource.Type]++
			}
		}
	}

	r.mu.Lock()
	snapshot.BridgeSubscriptions = len(r.bridgeSubs)
	r.mu.Unlock()
	return snapshot
}

func runtimeClosePhase(mode runtimeCloseMode) string {
	switch mode {
	case runtimeCloseModeUnmount:
		return "unmounted"
	case runtimeCloseModeShutdown:
		return "shutdown"
	default:
		return "closed"
	}
}
