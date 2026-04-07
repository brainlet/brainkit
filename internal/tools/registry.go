package tools

import (
	"fmt"
	"github.com/brainlet/brainkit/internal/syncx"

	"github.com/brainlet/brainkit/internal/sdkerrors"
)

// ToolRegistry manages all tools across the platform.
type ToolRegistry struct {
	mu    syncx.RWMutex
	tools map[string]*RegisteredTool
}

// New creates a new ToolRegistry.
func New() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*RegisteredTool),
	}
}

// Register adds a tool to the registry.
// For new-format names (containing "/"), Owner/Package/Version/ShortName are
// populated via ParseToolName if not already set.
// For bare names (no "/"), ShortName = name.
func (r *ToolRegistry) Register(tool RegisteredTool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tool.Name == "" {
		return fmt.Errorf("registry: tool name is required")
	}

	if IsNewFormat(tool.Name) {
		if tool.Owner == "" || tool.Package == "" || tool.Version == "" || tool.ShortName == "" {
			owner, pkg, version, short := ParseToolName(tool.Name)
			if tool.Owner == "" {
				tool.Owner = owner
			}
			if tool.Package == "" {
				tool.Package = pkg
			}
			if tool.Version == "" {
				tool.Version = version
			}
			if tool.ShortName == "" {
				tool.ShortName = short
			}
		}
	} else {
		// Bare short name (no "/") — just set ShortName.
		if tool.ShortName == "" {
			tool.ShortName = tool.Name
		}
	}

	r.tools[tool.Name] = &tool
	return nil
}

// Unregister removes a tool by its canonical name.
func (r *ToolRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// Resolve finds a tool by name with multi-level resolution.
//
// Resolution order:
//  1. Exact match
//  2. No owner -> default to "brainlet"
//  3. No version -> highest installed semver
//  4. Bare package/tool -> brainlet/{package}@{latest}/{tool}
//  5. Short name search across all tools
func (r *ToolRegistry) Resolve(name string) (*RegisteredTool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Level 1: Exact match
	if tool, ok := r.tools[name]; ok {
		return tool, nil
	}

	if IsNewFormat(name) {
		return r.resolveNewFormat(name)
	}

	// Level 5: Short name search across all tools
	for _, tool := range r.tools {
		if tool.ShortName == name {
			return tool, nil
		}
	}

	return nil, &sdkerrors.NotFoundError{Resource: "tool", Name: name}
}

// resolveNewFormat handles "/" separated names through levels 2-4.
func (r *ToolRegistry) resolveNewFormat(name string) (*RegisteredTool, error) {
	owner, pkg, version, tool := ParseToolName(name)

	// Level 2: No owner -> default to "brainlet"
	if owner == "" && pkg != "" && version != "" && tool != "" {
		candidate := ComposeName("brainlet", pkg, version, tool)
		if t, ok := r.tools[candidate]; ok {
			return t, nil
		}
		return nil, &sdkerrors.NotFoundError{Resource: "tool", Name: name}
	}

	// Level 3: No version -> find highest semver
	if owner != "" && pkg != "" && version == "" && tool != "" {
		if t := r.findHighestVersion(owner, pkg, tool); t != nil {
			return t, nil
		}
		return nil, &sdkerrors.NotFoundError{Resource: "tool", Name: name}
	}

	// Level 4: Bare package/tool (no owner, no version)
	if owner == "" && pkg != "" && version == "" && tool != "" {
		if t := r.findHighestVersion("brainlet", pkg, tool); t != nil {
			return t, nil
		}
		if t := r.findHighestVersionAnyOwner(pkg, tool); t != nil {
			return t, nil
		}
		return nil, &sdkerrors.NotFoundError{Resource: "tool", Name: name}
	}

	return nil, &sdkerrors.NotFoundError{Resource: "tool", Name: name}
}

func (r *ToolRegistry) findHighestVersion(owner, pkg, tool string) *RegisteredTool {
	var best *RegisteredTool
	for _, t := range r.tools {
		if t.Owner != owner || t.Package != pkg || t.ShortName != tool {
			continue
		}
		if IsPrerelease(t.Version) {
			continue
		}
		if best == nil || CompareSemver(t.Version, best.Version) > 0 {
			best = t
		}
	}
	return best
}

func (r *ToolRegistry) findHighestVersionAnyOwner(pkg, tool string) *RegisteredTool {
	var best *RegisteredTool
	for _, t := range r.tools {
		if t.Package != pkg || t.ShortName != tool {
			continue
		}
		if IsPrerelease(t.Version) {
			continue
		}
		if best == nil || CompareSemver(t.Version, best.Version) > 0 {
			best = t
		}
	}
	return best
}

// List returns tools, optionally filtered.
// Matches against Owner, Owner/Package, Package, or ShortName.
func (r *ToolRegistry) List(filter string) []RegisteredTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []RegisteredTool
	for _, tool := range r.tools {
		if filter == "" {
			result = append(result, *tool)
			continue
		}
		if tool.Owner == filter {
			result = append(result, *tool)
			continue
		}
		if tool.Owner != "" && tool.Package != "" && filter == tool.Owner+"/"+tool.Package {
			result = append(result, *tool)
			continue
		}
		if tool.Package == filter {
			result = append(result, *tool)
			continue
		}
		if tool.ShortName == filter {
			result = append(result, *tool)
			continue
		}
	}
	return result
}
