package registry

import (
	"fmt"
	"strings"
	"sync"
)

// ToolRegistry manages all tools across the platform.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*RegisteredTool
}

// New creates a new ToolRegistry.
func New() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*RegisteredTool),
	}
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(tool RegisteredTool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tool.Name == "" {
		return fmt.Errorf("registry: tool name is required")
	}
	if tool.ShortName == "" {
		_, tool.ShortName = ParseNamespace(tool.Name)
	}
	if tool.Namespace == "" {
		tool.Namespace, _ = ParseNamespace(tool.Name)
	}
	r.tools[tool.Name] = &tool
	return nil
}

// Unregister removes a tool.
func (r *ToolRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// Resolve finds a tool by name with namespace resolution.
// Fully qualified names are looked up directly.
// Short names are resolved: caller namespace → user → platform → plugin.*.
func (r *ToolRegistry) Resolve(name string, callerNamespace string) (*RegisteredTool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Exact match
	if tool, ok := r.tools[name]; ok {
		return tool, nil
	}

	// If fully qualified but not found
	if strings.Contains(name, ".") {
		return nil, fmt.Errorf("registry: tool %q not found", name)
	}

	// Short name resolution
	for _, ns := range ResolutionOrder(callerNamespace) {
		fullName := ns + "." + name
		if tool, ok := r.tools[fullName]; ok {
			return tool, nil
		}
	}

	// Plugin namespace fallback
	for _, tool := range r.tools {
		if tool.ShortName == name && strings.HasPrefix(tool.Namespace, "plugin.") {
			return tool, nil
		}
	}

	return nil, fmt.Errorf("registry: tool %q not found", name)
}

// List returns tools, optionally filtered by namespace.
func (r *ToolRegistry) List(namespaceFilter string) []RegisteredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []RegisteredTool
	for _, tool := range r.tools {
		if namespaceFilter == "" || tool.Namespace == namespaceFilter {
			result = append(result, *tool)
		}
	}
	return result
}
