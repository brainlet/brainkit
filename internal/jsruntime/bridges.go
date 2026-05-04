package jsruntime

import (
	js "github.com/brainlet/brainkit/internal/contract"
)

// registerBridges adds all Go bridge functions to the Kernel's QuickJS context.
// Each domain is registered in its own file for maintainability:
//   - bridges_request.go  — __go_brainkit_request, __go_brainkit_request_async
//   - bridges_control.go  — __go_brainkit_control (tools/agents/registry register/unregister)
//   - bridges_bus.go      — bus_send, bus_publish, bus_emit, bus_reply, subscribe, unsubscribe
//   - bridges_registry.go — registry resolve/runtime-resolve, has, list
//   - bridges_approval.go — __go_brainkit_await_approval
//   - bridges_scheduling.go — bus_schedule, bus_unschedule
//   - bridges_secrets.go  — __go_brainkit_secret_get
//   - bridges_util.go     — throwBrainkitError, redactCredentials, console_log_tagged
func (r *Runtime) registerBridges() {
	qctx := r.bridge.Context()

	r.registerRequestBridges(qctx, r.bus)
	r.registerControlBridges(qctx)
	r.registerBusBridges(qctx)
	r.registerLoggingBridge(qctx)
	r.registerRegistryBridges(qctx)
	r.registerApprovalBridges(qctx)
	r.registerSchedulingBridges(qctx)
	r.registerSecretBridges(qctx)

	// Set context globals
	qctx.Globals().Set(js.JSSandboxID, qctx.NewString(r.agents.ID()))
	qctx.Globals().Set(js.JSSandboxNamespace, qctx.NewString(r.core.Namespace()))
	qctx.Globals().Set(js.JSSandboxCallerID, qctx.NewString(r.core.CallerID()))
}
