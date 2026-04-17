package gateway

import "github.com/brainlet/brainkit"

// Name reports the module identifier.
func (gw *Gateway) Name() string { return "gateway" }

// Status reports maturity (stable).
func (gw *Gateway) Status() brainkit.ModuleStatus { return brainkit.ModuleStatusStable }

// Init captures the Kit as the gateway's runtime and starts the HTTP server.
// Routes registered via Handle / HandleStream / HandleWebSocket / HandleWebhook
// before Init are installed at startup; routes added after Init update the
// live route table.
func (gw *Gateway) Init(k *brainkit.Kit) error {
	gw.SetRuntime(k)
	return gw.Start()
}

// Close stops the HTTP server and unsubscribes bus route commands.
func (gw *Gateway) Close() error {
	return gw.Stop()
}
