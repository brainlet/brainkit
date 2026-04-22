package brainkit

import (
	"fmt"
	"sort"
	"sync"
)

// ModuleContext is what the server hands to a factory at Build time.
// Kit doesn't exist yet during Build — the factory's only job is to
// decode its config section and return a Module struct. The Module's
// Init(k *Kit) does the real Kit-facing wiring once all modules are
// attached.
type ModuleContext struct {
	// FSRoot is the server's sandbox root. Factories use it for
	// default paths (e.g. "<FSRoot>/audit.db" when config omits Path).
	FSRoot string

	// Decode unmarshals the YAML subtree under `modules.<name>` into
	// the factory's typed config. Pass a pointer:
	//
	//	var cfg Config
	//	if err := ctx.Decode(&cfg); err != nil { return nil, err }
	//
	// Backend-agnostic: the server layer picks the encoding (yaml,
	// json, toml) — the registry never sees it.
	Decode func(any) error
}

// ModuleFactory constructs a Module from a config section. Third-party
// binaries register factories in their package init() so a plain
// `import _ "example.com/acme/brainkit/billing"` in a custom brainkit
// binary is enough to make `modules.billing:` work in server YAML.
//
//	type Config struct {
//	    Path string `yaml:"path"`
//	}
//	func (f *Factory) Build(ctx brainkit.ModuleContext) (brainkit.Module, error) {
//	    var cfg Config
//	    if err := ctx.Decode(&cfg); err != nil {
//	        return nil, err
//	    }
//	    if cfg.Path == "" {
//	        cfg.Path = filepath.Join(ctx.FSRoot, "billing.db")
//	    }
//	    return newModule(cfg), nil
//	}
type ModuleFactory interface {
	// Build constructs the module. Returning (nil, nil) is allowed
	// when the config section is present but empty and the factory
	// opts to skip the module.
	Build(ctx ModuleContext) (Module, error)
}

// ModuleDescriptor is optional metadata a factory can expose for
// `brainkit modules list` / docs. Implement ModuleDescriber on the
// factory type to advertise it.
type ModuleDescriptor struct {
	Name    string       `json:"name"`
	Status  ModuleStatus `json:"status,omitempty"`
	Summary string       `json:"summary,omitempty"`
}

// ModuleDescriber is an optional interface a ModuleFactory can
// implement to surface its descriptor to the CLI.
type ModuleDescriber interface {
	Describe() ModuleDescriptor
}

// ModuleFactoryFunc adapts a plain function to the ModuleFactory
// interface. Convenient for factories that don't need extra state.
type ModuleFactoryFunc func(ctx ModuleContext) (Module, error)

// Build satisfies ModuleFactory.
func (f ModuleFactoryFunc) Build(ctx ModuleContext) (Module, error) { return f(ctx) }

var (
	moduleRegistryMu sync.RWMutex
	moduleRegistry   = map[string]ModuleFactory{}
)

// RegisterModule adds a factory to the global module registry under
// `name`. Call from package init() so a blank-import wires the module
// in before LoadConfig runs.
//
// Double registration panics — the common case is a copy-paste bug or
// two versions of the same package linked into one binary, both of
// which are bugs worth surfacing loudly at startup.
func RegisterModule(name string, factory ModuleFactory) {
	if name == "" {
		panic("brainkit: RegisterModule: name is required")
	}
	if factory == nil {
		panic(fmt.Sprintf("brainkit: RegisterModule(%q): factory is nil", name))
	}
	moduleRegistryMu.Lock()
	defer moduleRegistryMu.Unlock()
	if _, exists := moduleRegistry[name]; exists {
		panic(fmt.Sprintf("brainkit: RegisterModule(%q): already registered", name))
	}
	moduleRegistry[name] = factory
}

// LookupModuleFactory returns the factory registered under name, or
// (nil, false) if none is.
func LookupModuleFactory(name string) (ModuleFactory, bool) {
	moduleRegistryMu.RLock()
	defer moduleRegistryMu.RUnlock()
	f, ok := moduleRegistry[name]
	return f, ok
}

// RegisteredModuleNames returns the sorted list of module names
// currently in the registry. Used by `brainkit modules list` and by
// server-side YAML processing (to reject unknown keys with a helpful
// "did you mean …" hint).
func RegisteredModuleNames() []string {
	moduleRegistryMu.RLock()
	defer moduleRegistryMu.RUnlock()
	names := make([]string, 0, len(moduleRegistry))
	for name := range moduleRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RegisteredModules returns descriptors for every registered module.
// Factories that don't implement ModuleDescriber get a minimal
// descriptor with just Name.
func RegisteredModules() []ModuleDescriptor {
	moduleRegistryMu.RLock()
	defer moduleRegistryMu.RUnlock()
	out := make([]ModuleDescriptor, 0, len(moduleRegistry))
	for name, f := range moduleRegistry {
		desc := ModuleDescriptor{Name: name}
		if d, ok := f.(ModuleDescriber); ok {
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

// unregisterModuleForTest removes a factory from the registry. Test
// helper — not part of the public surface.
func unregisterModuleForTest(name string) {
	moduleRegistryMu.Lock()
	defer moduleRegistryMu.Unlock()
	delete(moduleRegistry, name)
}
