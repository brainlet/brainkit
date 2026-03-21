package plugin

import (
	"context"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

// Bridge is the narrow interface the plugin manager needs from the Kit.
type Bridge interface {
	Bus() *bus.Bus
	KitName() string
	Tools() *registry.ToolRegistry
	Deploy(ctx context.Context, source, code string) error
	Teardown(ctx context.Context, source string) error
}
