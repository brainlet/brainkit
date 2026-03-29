package workflow

import (
	"sync"
)

// HostFunctionRegistry manages dynamically registered host functions.
// Plugins register host functions at startup. Workflows use them.
type HostFunctionRegistry struct {
	mu        sync.RWMutex
	functions map[string]map[string]*HostFunctionDef // module → name → def
}

// NewHostFunctionRegistry creates an empty registry.
func NewHostFunctionRegistry() *HostFunctionRegistry {
	return &HostFunctionRegistry{
		functions: make(map[string]map[string]*HostFunctionDef),
	}
}

// Register adds a host function to the registry.
func (r *HostFunctionRegistry) Register(def HostFunctionDef) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.functions[def.Module]; !ok {
		r.functions[def.Module] = make(map[string]*HostFunctionDef)
	}
	r.functions[def.Module][def.Name] = &def
}

// Unregister removes a host function.
func (r *HostFunctionRegistry) Unregister(module, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if mod, ok := r.functions[module]; ok {
		delete(mod, name)
		if len(mod) == 0 {
			delete(r.functions, module)
		}
	}
}

// UnregisterModule removes all host functions from a module.
func (r *HostFunctionRegistry) UnregisterModule(module string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.functions, module)
}

// Get returns a host function definition, or nil if not found.
func (r *HostFunctionRegistry) Get(module, name string) *HostFunctionDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if mod, ok := r.functions[module]; ok {
		return mod[name]
	}
	return nil
}

// ListModules returns all registered module names.
func (r *HostFunctionRegistry) ListModules() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modules := make([]string, 0, len(r.functions))
	for mod := range r.functions {
		modules = append(modules, mod)
	}
	return modules
}

// ListFunctions returns all host functions in a module.
func (r *HostFunctionRegistry) ListFunctions(module string) []*HostFunctionDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mod, ok := r.functions[module]
	if !ok {
		return nil
	}
	result := make([]*HostFunctionDef, 0, len(mod))
	for _, def := range mod {
		result = append(result, def)
	}
	return result
}

// All returns all registered host functions across all modules.
func (r *HostFunctionRegistry) All() []*HostFunctionDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*HostFunctionDef
	for _, mod := range r.functions {
		for _, def := range mod {
			result = append(result, def)
		}
	}
	return result
}
