package brainkit

import (
	js "github.com/brainlet/brainkit/internal/contract"
)

// registerBridges adds all Go bridge functions to the Kernel's QuickJS context.
// Each domain is registered in its own file for maintainability:
//   - bridges_request.go  — __go_brainkit_request, __go_brainkit_request_async
//   - bridges_control.go  — __go_brainkit_control (tools/agents/registry register/unregister)
//   - bridges_bus.go      — bus_send, bus_publish, bus_emit, bus_reply, subscribe, unsubscribe
//   - bridges_registry.go — __go_registry_resolve, __go_registry_has, __go_registry_list
//   - bridges_approval.go — __go_brainkit_await_approval
//   - bridges_scheduling.go — bus_schedule, bus_unschedule
//   - bridges_secrets.go  — __go_brainkit_secret_get
//   - bridges_util.go     — throwBrainkitError, redactCredentials, console_log_tagged
func (k *Kernel) registerBridges() {
	qctx := k.bridge.Context()
	invoker := newLocalInvoker(k)

	k.registerRequestBridges(qctx, invoker)
	k.registerControlBridges(qctx)
	k.registerBusBridges(qctx)
	k.registerLoggingBridge(qctx)
	k.registerRegistryBridges(qctx)
	k.registerApprovalBridges(qctx)
	k.registerSchedulingBridges(qctx)
	k.registerSecretBridges(qctx)

	// Set context globals
	qctx.Globals().Set(js.JSSandboxID, qctx.NewString(k.agents.ID()))
	qctx.Globals().Set(js.JSSandboxNamespace, qctx.NewString(k.namespace))
	qctx.Globals().Set(js.JSSandboxCallerID, qctx.NewString(k.callerID))
}
