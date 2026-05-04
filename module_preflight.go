package brainkit

import (
	"fmt"
	"sort"
	"strings"

	bkmodule "github.com/brainlet/brainkit/module"
)

func (k *Kit) preflightModuleMount(desc bkmodule.Descriptor) error {
	preflight := k.preflightModuleDescriptor(desc)
	if preflight.Ready {
		return nil
	}
	name := desc.Name
	if name == "" {
		name = "unknown"
	}
	if len(preflight.MissingRequiredCapabilities) > 0 && len(preflight.MissingModules) == 0 && len(preflight.Errors) == 0 {
		return fmt.Errorf("brainkit: module %q missing required capabilities before mount: %s", name, strings.Join(formatPreflightCapabilities(name, preflight.MissingRequiredCapabilities), ", "))
	}
	var parts []string
	if len(preflight.MissingModules) > 0 {
		parts = append(parts, "missing modules: "+strings.Join(preflight.MissingModules, ", "))
	}
	if len(preflight.MissingRequiredCapabilities) > 0 {
		parts = append(parts, "missing required capabilities: "+strings.Join(formatPreflightCapabilities(name, preflight.MissingRequiredCapabilities), ", "))
	}
	if len(preflight.Errors) > 0 {
		parts = append(parts, "errors: "+strings.Join(preflight.Errors, ", "))
	}
	return fmt.Errorf("brainkit: module %q preflight failed before mount: %s", name, strings.Join(parts, "; "))
}

func formatPreflightCapabilities(root string, caps []bkmodule.CapabilityAvailability) []string {
	out := make([]string, 0, len(caps))
	for _, cap := range caps {
		label := formatRequiredCapability(bkmodule.CapabilityDescriptor{
			Name: cap.Name,
			Type: cap.Type,
		})
		if cap.Module != "" && cap.Module != root {
			label = cap.Module + ": " + label
		}
		out = append(out, label)
	}
	sort.Strings(out)
	return out
}

func (k *Kit) preflightModuleDescriptor(desc bkmodule.Descriptor) bkmodule.ModulePreflight {
	desc = bkmodule.NormalizeDescriptor(desc.Name, desc)
	planner := &modulePreflightPlanner{
		k:                   k,
		plannedCapabilities: map[string]string{},
		visiting:            map[string]struct{}{},
		visited:             map[string]bool{},
		requiredModules:     map[string]bkmodule.ModuleDependencyStatus{},
		missingModules:      map[string]struct{}{},
		requiredCaps:        map[string]bkmodule.CapabilityAvailability{},
		optionalCaps:        map[string]bkmodule.CapabilityAvailability{},
		missingCaps:         map[string]bkmodule.CapabilityAvailability{},
		errors:              map[string]struct{}{},
	}
	planner.check(desc)
	return planner.result()
}

type modulePreflightPlanner struct {
	k                   *Kit
	plannedCapabilities map[string]string
	visiting            map[string]struct{}
	stack               []string
	visited             map[string]bool

	requiredModules map[string]bkmodule.ModuleDependencyStatus
	missingModules  map[string]struct{}
	requiredCaps    map[string]bkmodule.CapabilityAvailability
	optionalCaps    map[string]bkmodule.CapabilityAvailability
	missingCaps     map[string]bkmodule.CapabilityAvailability
	errors          map[string]struct{}
}

func (p *modulePreflightPlanner) check(desc bkmodule.Descriptor) bool {
	desc = bkmodule.NormalizeDescriptor(desc.Name, desc)
	id := desc.Name
	if id == "" {
		id = "unknown"
	}
	if ready, ok := p.visited[id]; ok {
		return ready
	}
	if _, ok := p.visiting[id]; ok {
		p.addError("cyclic module dependency: " + p.cycleLabel(id))
		return false
	}
	p.visiting[id] = struct{}{}
	p.stack = append(p.stack, id)
	ready := true

	for _, dep := range desc.Requires {
		if dep == "" {
			continue
		}
		if dep == id {
			p.addError(fmt.Sprintf("module %q cannot require itself", id))
			ready = false
			continue
		}
		status := p.dependencyStatus(id, dep)
		p.requiredModules[dependencyStatusKey(status)] = status
		if !status.Available {
			p.missingModules[dep] = struct{}{}
			ready = false
			continue
		}
		if status.Mounted {
			continue
		}
		depDesc, err := registeredModuleDescriptor(dep)
		if err != nil {
			p.missingModules[dep] = struct{}{}
			ready = false
			continue
		}
		if !p.check(depDesc) {
			ready = false
		}
	}

	for _, cap := range desc.Capabilities {
		if cap.Name == "" {
			continue
		}
		switch cap.Direction {
		case bkmodule.CapabilityRequired:
			availability := p.capabilityAvailability(id, cap)
			p.requiredCaps[capabilityAvailabilityKey(availability)] = availability
			if !availability.Available {
				p.missingCaps[capabilityAvailabilityKey(availability)] = availability
				ready = false
			}
		case bkmodule.CapabilityOptional:
			availability := p.capabilityAvailability(id, cap)
			p.optionalCaps[capabilityAvailabilityKey(availability)] = availability
		}
	}

	if ready {
		for _, cap := range desc.Capabilities {
			if cap.Name == "" || cap.Direction != bkmodule.CapabilityProvided {
				continue
			}
			p.plannedCapabilities[cap.Name] = id
		}
	}

	delete(p.visiting, id)
	p.stack = p.stack[:len(p.stack)-1]
	p.visited[id] = ready
	return ready
}

func (p *modulePreflightPlanner) dependencyStatus(requestedBy, name string) bkmodule.ModuleDependencyStatus {
	_, mounted := p.k.Module(name)
	_, registered := bkmodule.Lookup(name)
	return bkmodule.ModuleDependencyStatus{
		Name:        name,
		RequestedBy: requestedBy,
		Mounted:     mounted,
		Registered:  registered,
		Available:   mounted || registered,
	}
}

func (p *modulePreflightPlanner) capabilityAvailability(moduleName string, cap bkmodule.CapabilityDescriptor) bkmodule.CapabilityAvailability {
	source, provider, available := p.capabilitySource(cap.Name)
	return bkmodule.CapabilityAvailability{
		Module:    moduleName,
		Name:      cap.Name,
		Direction: cap.Direction,
		Type:      cap.Type,
		Summary:   cap.Summary,
		Available: available,
		Source:    source,
		Provider:  provider,
	}
}

func (p *modulePreflightPlanner) capabilitySource(name string) (source, provider string, available bool) {
	if provider, ok := p.plannedCapabilities[name]; ok {
		return "planned", provider, true
	}
	return p.k.capabilitySource(name)
}

func (p *modulePreflightPlanner) cycleLabel(id string) string {
	start := 0
	for i, name := range p.stack {
		if name == id {
			start = i
			break
		}
	}
	cycle := append([]string(nil), p.stack[start:]...)
	cycle = append(cycle, id)
	return strings.Join(cycle, " -> ")
}

func (p *modulePreflightPlanner) addError(msg string) {
	if msg != "" {
		p.errors[msg] = struct{}{}
	}
}

func (p *modulePreflightPlanner) result() bkmodule.ModulePreflight {
	out := bkmodule.ModulePreflight{
		RequiredModules:             sortedDependencyStatuses(p.requiredModules),
		MissingModules:              sortedStringSet(p.missingModules),
		RequiredCapabilities:        sortedCapabilityAvailabilities(p.requiredCaps),
		OptionalCapabilities:        sortedCapabilityAvailabilities(p.optionalCaps),
		MissingRequiredCapabilities: sortedCapabilityAvailabilities(p.missingCaps),
		Errors:                      sortedStringSet(p.errors),
	}
	out.Ready = len(out.MissingModules) == 0 && len(out.MissingRequiredCapabilities) == 0 && len(out.Errors) == 0
	return out
}

func dependencyStatusKey(status bkmodule.ModuleDependencyStatus) string {
	return status.RequestedBy + "\x00" + status.Name
}

func capabilityAvailabilityKey(cap bkmodule.CapabilityAvailability) string {
	return cap.Module + "\x00" + cap.Name
}

func sortedDependencyStatuses(in map[string]bkmodule.ModuleDependencyStatus) []bkmodule.ModuleDependencyStatus {
	out := make([]bkmodule.ModuleDependencyStatus, 0, len(in))
	for _, status := range in {
		out = append(out, status)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RequestedBy != out[j].RequestedBy {
			return out[i].RequestedBy < out[j].RequestedBy
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func sortedCapabilityAvailabilities(in map[string]bkmodule.CapabilityAvailability) []bkmodule.CapabilityAvailability {
	out := make([]bkmodule.CapabilityAvailability, 0, len(in))
	for _, cap := range in {
		out = append(out, cap)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Module != out[j].Module {
			return out[i].Module < out[j].Module
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func sortedStringSet(in map[string]struct{}) []string {
	out := make([]string, 0, len(in))
	for value := range in {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
