package engine

import (
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modulehost/transporthost"
)

// TransportDebugSnapshot is a test/debug view of transport lifecycle state.
type TransportDebugSnapshot = transporthost.DebugSnapshot

// TransportDebugSnapshot returns transport-host bookkeeping counts for
// lifecycle tests.
func (k *Kernel) TransportDebugSnapshot() TransportDebugSnapshot {
	if k == nil || k.transportHost == nil {
		return TransportDebugSnapshot{}
	}
	return k.transportHost.DebugSnapshot()
}

// LifecycleDebugSnapshot returns Brainkit-owned lifecycle counters for
// operator inspect/debug surfaces.
func (k *Kernel) LifecycleDebugSnapshot() bkmodule.LifecycleRuntimeDebug {
	if k == nil {
		return bkmodule.LifecycleRuntimeDebug{}
	}
	var provider bkmodule.LifecycleProviderDebug
	if k.providerHost != nil {
		snap := k.providerHost.DebugSnapshot()
		provider = bkmodule.LifecycleProviderDebug{
			AIProviders:      snap.AIProviders,
			VectorStores:     snap.VectorStores,
			Storages:         snap.Storages,
			Closing:          snap.Closing,
			Closed:           snap.Closed,
			ActiveProbes:     snap.ActiveProbes,
			ActiveOperations: snap.ActiveOperations,
		}
	}
	var storage bkmodule.LifecycleStorageDebug
	if k.storageHost != nil {
		snap := k.storageHost.DebugSnapshot()
		storage = bkmodule.LifecycleStorageDebug{
			Closing:      snap.Closing,
			BridgeCount:  snap.BridgeCount,
			BridgeNames:  append([]string(nil), snap.BridgeNames...),
			ActiveCloses: snap.ActiveCloses,
		}
	}
	var transport bkmodule.LifecycleTransportDebug
	if k.transportHost != nil {
		snap := k.transportHost.DebugSnapshot()
		transport = bkmodule.LifecycleTransportDebug{
			Kind:                   k.transportHost.TransportKind(),
			OwnsTransport:          snap.OwnsTransport,
			ActiveSubscriptions:    snap.ActiveSubscriptions,
			ActiveStreamHeartbeats: k.streamTracker.Active(),
			CallerClosed:           snap.Caller.Closed,
			CallerPendingCalls:     snap.Caller.PendingCalls,
			CallerStreamDrains:     snap.Caller.ActiveStreamDrains,
			ClosingRouter:          snap.ClosingRouter,
			ClosingCaller:          snap.ClosingCaller,
			ClosingTransport:       snap.ClosingTransport,
			ClosedRouter:           snap.ClosedRouter,
			ClosedCaller:           snap.ClosedCaller,
			ClosedTransport:        snap.ClosedTransport,
			Router: bkmodule.LifecycleRouterDebug{
				Handlers:        snap.Router.Handlers,
				StartedHandlers: snap.Router.StartedHandlers,
				StoppedHandlers: snap.Router.StoppedHandlers,
				Topics:          snap.Router.Topics,
			},
		}
	}
	return bkmodule.LifecycleRuntimeDebug{
		RuntimeID:      k.config.RuntimeID,
		Namespace:      k.namespace,
		CallerID:       k.callerID,
		ActiveHandlers: k.activeHandlers.Load(),
		Draining:       k.IsDraining(),
		Provider:       provider,
		Storage:        storage,
		Transport:      transport,
	}
}
