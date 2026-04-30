// Package module defines Brainkit's hot-mountable module authoring API.
package module

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"sync"
)

// Module is a hot-mountable capability bundle.
type Module interface {
	ID() string
	Mount(context.Context, Host) error
}

// Status reports a module's maturity for CLI/docs surfaces.
type Status string

const (
	StatusStable Status = "stable"
	StatusBeta   Status = "beta"
	StatusWIP    Status = "wip"
)

// Descriptor is the module manifest used for listing, docs, dependency
// resolution, and runtime introspection.
type Descriptor struct {
	Name          string                 `json:"name"`
	Status        Status                 `json:"status,omitempty"`
	Summary       string                 `json:"summary,omitempty"`
	Provides      []string               `json:"provides,omitempty"`
	Requires      []string               `json:"requires,omitempty"`
	Commands      []MessageDescriptor    `json:"commands,omitempty"`
	Events        []MessageDescriptor    `json:"events,omitempty"`
	Subscriptions []MessageDescriptor    `json:"subscriptions,omitempty"`
	Capabilities  []CapabilityDescriptor `json:"capabilities,omitempty"`
	Resources     []ResourceDescriptor   `json:"resources,omitempty"`
}

// Describer is implemented by factories or modules that expose metadata.
type Describer interface {
	Describe() Descriptor
}

// StatusReporter is implemented by modules that expose a maturity tag.
type StatusReporter interface {
	Status() Status
}

// Host is the surface a module can use during Mount.
type Host interface {
	Scope() Scope
	Messages() MessageHost
	Commands() CommandHost
	Tools() ToolHost
	Capabilities() CapabilityHost
	Logger() *slog.Logger
}

// BuildContext is what assembly layers hand to a factory before a Kit exists.
type BuildContext struct {
	FSRoot string
	Decode func(any) error
}

// Factory constructs a Module from a config section.
type Factory interface {
	Build(BuildContext) (Module, error)
}

// FactoryFunc adapts a plain function to Factory.
type FactoryFunc func(BuildContext) (Module, error)

// Build satisfies Factory.
func (f FactoryFunc) Build(ctx BuildContext) (Module, error) { return f(ctx) }

// ModuleBuildConfig carries runtime module factory input. JSON is decoded as
// YAML-compatible structured data so module YAML tags still apply; YAML is
// useful for operators and CLIs that want to pass a config file verbatim.
type ModuleBuildConfig struct {
	JSON json.RawMessage `json:"json,omitempty"`
	YAML string          `json:"yaml,omitempty"`
}

// ModuleLifecycle is the control-plane capability for registered linked-code
// module lifecycle operations.
type ModuleLifecycle interface {
	MountModule(context.Context, string, ModuleBuildConfig) (Descriptor, error)
	UnmountModule(context.Context, string) (Descriptor, error)
	DescribeModule(context.Context, string) (Descriptor, bool, error)
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Factory{}
)

// Register adds a factory to the process registry.
func Register(name string, factory Factory) {
	if name == "" {
		panic("brainkit/module: Register: name is required")
	}
	if factory == nil {
		panic(fmt.Sprintf("brainkit/module: Register(%q): factory is nil", name))
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("brainkit/module: Register(%q): already registered", name))
	}
	registry[name] = factory
}

// Lookup returns the factory registered under name.
func Lookup(name string) (Factory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := registry[name]
	return f, ok
}

// Names returns registered module names in stable order.
func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Descriptors returns descriptors for every registered module.
func Descriptors() []Descriptor {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Descriptor, 0, len(registry))
	for name, f := range registry {
		desc := Descriptor{Name: name}
		if d, ok := f.(Describer); ok {
			desc = d.Describe()
			if desc.Name == "" {
				desc.Name = name
			}
		}
		out = append(out, NormalizeDescriptor(name, desc))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
