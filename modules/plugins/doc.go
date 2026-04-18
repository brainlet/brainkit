// Package plugins is the brainkit.Module form of subprocess plugins.
//
// Plugins are separately-compiled binaries that connect back to the Kit
// over a localhost WebSocket endpoint and register tools + bus
// subscriptions. The module owns three pieces:
//
//  1. the plugin subprocess manager (launch / restart / shutdown with
//     exponential backoff and optional persistence to a Store for restart
//     recovery),
//  2. the WebSocket server plugins dial into (manifest handshake, tool
//     call dispatch, bus publish / subscribe bridging, heartbeat), and
//  3. the plugin.* bus command surface (plugin.start, plugin.stop,
//     plugin.restart, plugin.listRunning, plugin.status, plugin.manifest).
//
// Plugins require a non-memory transport — the WebSocket server needs
// real networking and the bus commands run over the external transport
// the plugins share with the host Kit. Without the module, the plugin.*
// commands are absent and no subprocess is launched.
//
// Usage:
//
//	kit, _ := brainkit.New(brainkit.Config{
//	    Transport: brainkit.EmbeddedNATS(),
//	    Modules: []brainkit.Module{
//	        plugins.NewModule(plugins.Config{
//	            Plugins: []brainkit.PluginConfig{{Name: "foo", Binary: "./foo"}},
//	            Store:   kitStore, // optional, enables restart recovery
//	        }),
//	    },
//	})
package plugins
