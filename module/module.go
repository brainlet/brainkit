// Package module defines Brainkit's hot-mountable module authoring API.
package module

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/brainlet/brainkit/sdk"
)

// Module is a hot-mountable capability bundle.
type Module interface {
	ID() string
	Mount(context.Context, Host) error
}

// DependencyReporter is optionally implemented by modules that need other
// modules mounted first. The host resolves registered dependencies before
// mounting the module.
type DependencyReporter interface {
	Dependencies() []string
}

// Status reports a module's maturity for CLI/docs surfaces.
type Status string

const (
	StatusStable Status = "stable"
	StatusBeta   Status = "beta"
	StatusWIP    Status = "wip"
)

// Descriptor is optional module metadata for listing and docs.
type Descriptor struct {
	Name     string   `json:"name"`
	Status   Status   `json:"status,omitempty"`
	Summary  string   `json:"summary,omitempty"`
	Provides []string `json:"provides,omitempty"`
	Requires []string `json:"requires,omitempty"`
}

// Describer is implemented by factories or modules that expose metadata.
type Describer interface {
	Describe() Descriptor
}

// Host is the surface a module can use during Mount.
type Host interface {
	Scope() Scope
	Runtime() sdk.Runtime
	Caller() *sdk.Caller
	Messages() MessageHost
	Commands() CommandHost
	Tools() ToolHost
	Capabilities() CapabilityHost
	Logger() *slog.Logger
	Store() any
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
		out = append(out, desc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
