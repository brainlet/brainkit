package engine

import "github.com/brainlet/brainkit/modulecap/runtime"

// JSRuntimeDebugSnapshot returns lifecycle counters for the attached JS
// runtime, when present.
func (k *Kernel) JSRuntimeDebugSnapshot() runtimecap.DebugSnapshot {
	if k == nil {
		return runtimecap.DebugSnapshot{Phase: "absent", Closed: true}
	}
	k.mu.Lock()
	rt := k.jsRuntime
	k.mu.Unlock()
	if rt == nil {
		return runtimecap.DebugSnapshot{Phase: "absent", Closed: true}
	}
	var runtimeHostSubscriptions int
	if k.runtimeHost != nil {
		runtimeHostSubscriptions = k.runtimeHost.DebugSnapshot().PropagationSubscriptions
	}
	if debug, ok := rt.(runtimecap.DebugSnapshotter); ok {
		snapshot := debug.JSRuntimeDebugSnapshot()
		snapshot.RuntimeHostPropagationSubscriptions = runtimeHostSubscriptions
		return snapshot
	}
	snapshot := runtimecap.DebugSnapshot{Phase: "active"}
	snapshot.RuntimeHostPropagationSubscriptions = runtimeHostSubscriptions
	deployments := rt.ListDeployments()
	snapshot.ActiveDeployments = len(deployments)
	resources, _ := rt.ListResources()
	snapshot.ResourceCount = len(resources)
	if len(resources) > 0 {
		snapshot.ResourcesByType = make(map[string]int)
		for _, resource := range resources {
			snapshot.ResourcesByType[resource.Type]++
		}
	}
	return snapshot
}
