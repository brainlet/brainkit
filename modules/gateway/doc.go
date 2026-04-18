// Package gateway exposes a brainkit Kit over HTTP. Routes map
// incoming requests onto bus topics; responses flow back through the
// shared-inbox Caller.
//
// Routes are registered via Handle / HandleStream / HandleWebSocket /
// HandleWebhook on *Gateway, or over the gateway.routes.* bus
// commands (installed at module init). Built-in health endpoints
// publish KitHealthMsg on /health unless disabled via Config.NoHealth.
//
// Status: stable.
package gateway
