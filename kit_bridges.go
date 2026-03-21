package brainkit

// Bridge adapters that make *Kit satisfy the narrow interfaces
// required by internal/wasm and internal/plugin.

import (
	"context"

	"github.com/brainlet/brainkit/bus"
	iplugin "github.com/brainlet/brainkit/internal/plugin"
	iwasm "github.com/brainlet/brainkit/internal/wasm"
	"github.com/brainlet/brainkit/registry"
)

// ---------------------------------------------------------------------------
// WASM bridge — satisfies wasm.BusBridge
// ---------------------------------------------------------------------------

// kitBusBridge adapts *Kit to the wasm.BusBridge interface.
// Kit has Bus as a public field (not a method), so we can't satisfy the
// interface directly — this thin wrapper provides the method set.
type kitBusBridge struct {
	kit *Kit
}

var _ iwasm.BusBridge = (*kitBusBridge)(nil)

func (b *kitBusBridge) Bus() *bus.Bus            { return b.kit.Bus }
func (b *kitBusBridge) CallerID() string         { return b.kit.callerID }
func (b *kitBusBridge) WASMStore() iwasm.Store   { return b.kit.config.Store }
func (b *kitBusBridge) WASMBundleSource() string { return wasmBundleSource }

// ---------------------------------------------------------------------------
// Plugin bridge — satisfies plugin.Bridge
// ---------------------------------------------------------------------------

// kitPluginBridge adapts *Kit to the plugin.Bridge interface.
type kitPluginBridge struct {
	kit *Kit
}

var _ iplugin.Bridge = (*kitPluginBridge)(nil)

func (b *kitPluginBridge) Bus() *bus.Bus                 { return b.kit.Bus }
func (b *kitPluginBridge) KitName() string               { return b.kit.config.Name }
func (b *kitPluginBridge) Tools() *registry.ToolRegistry { return b.kit.Tools }

func (b *kitPluginBridge) Deploy(ctx context.Context, source, code string) error {
	_, err := b.kit.Deploy(ctx, source, code)
	return err
}

func (b *kitPluginBridge) Teardown(ctx context.Context, source string) error {
	_, err := b.kit.Teardown(ctx, source)
	return err
}
