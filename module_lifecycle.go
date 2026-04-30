package brainkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	bkmodule "github.com/brainlet/brainkit/module"
	"gopkg.in/yaml.v3"
)

type kitModuleLifecycle struct {
	k *Kit
}

func (l kitModuleLifecycle) MountModule(ctx context.Context, id string, cfg bkmodule.ModuleBuildConfig) (bkmodule.Descriptor, error) {
	if id == "" {
		return bkmodule.Descriptor{}, fmt.Errorf("brainkit: module ID is required")
	}
	mod, err := buildRegisteredModuleWithConfig(id, l.k.fsRoot, cfg)
	if err != nil {
		return bkmodule.Descriptor{}, err
	}
	if err := l.k.Mount(ctx, mod); err != nil {
		return bkmodule.Descriptor{}, err
	}
	desc, ok := l.k.mountedModuleDescriptor(id)
	if !ok {
		return bkmodule.Descriptor{}, fmt.Errorf("brainkit: module %q mounted without descriptor", id)
	}
	return desc, nil
}

func (l kitModuleLifecycle) UnmountModule(ctx context.Context, id string) (bkmodule.Descriptor, error) {
	if id == "" {
		return bkmodule.Descriptor{}, fmt.Errorf("brainkit: module ID is required")
	}
	if dependents := l.k.mountedDependents(id); len(dependents) > 0 {
		return bkmodule.Descriptor{}, fmt.Errorf("brainkit: module %q is required by mounted module(s): %s", id, strings.Join(dependents, ", "))
	}
	desc, ok := l.k.mountedModuleDescriptor(id)
	if !ok {
		return bkmodule.Descriptor{}, fmt.Errorf("brainkit: module %q is not mounted", id)
	}
	if err := l.k.Unmount(ctx, id); err != nil {
		return bkmodule.Descriptor{}, err
	}
	return desc, nil
}

func (l kitModuleLifecycle) DescribeModule(_ context.Context, id string) (bkmodule.Descriptor, bool, error) {
	if id == "" {
		return bkmodule.Descriptor{}, false, fmt.Errorf("brainkit: module ID is required")
	}
	if desc, ok := l.k.mountedModuleDescriptor(id); ok {
		return desc, true, nil
	}
	desc, err := registeredModuleDescriptor(id)
	if err != nil {
		return bkmodule.Descriptor{}, false, err
	}
	return desc, false, nil
}

func (k *Kit) mountedModuleDescriptor(id string) (bkmodule.Descriptor, bool) {
	k.mountMu.Lock()
	defer k.mountMu.Unlock()
	desc, ok := k.descs[id]
	return desc, ok
}

func (k *Kit) mountedDependents(id string) []string {
	k.mountMu.Lock()
	defer k.mountMu.Unlock()
	var out []string
	for name, desc := range k.descs {
		if name == id {
			continue
		}
		for _, dep := range desc.Requires {
			if dep == id {
				out = append(out, name)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

func registeredModuleDescriptor(id string) (bkmodule.Descriptor, error) {
	factory, ok := bkmodule.Lookup(id)
	if !ok {
		return bkmodule.Descriptor{}, fmt.Errorf("brainkit: module %q is not registered", id)
	}
	desc := bkmodule.Descriptor{Name: id}
	if describer, ok := factory.(bkmodule.Describer); ok {
		desc = describer.Describe()
	}
	return bkmodule.NormalizeDescriptor(id, desc), nil
}

func buildRegisteredModule(id, fsRoot string) (bkmodule.Module, error) {
	return buildRegisteredModuleWithConfig(id, fsRoot, bkmodule.ModuleBuildConfig{})
}

func buildRegisteredModuleWithConfig(id, fsRoot string, cfg bkmodule.ModuleBuildConfig) (bkmodule.Module, error) {
	factory, ok := bkmodule.Lookup(id)
	if !ok {
		if id == "jsruntime" {
			return nil, fmt.Errorf("brainkit: JS runtime requested but module %q is not registered; import github.com/brainlet/brainkit/modules/jsruntime or github.com/brainlet/brainkit/presets/standard", id)
		}
		return nil, fmt.Errorf("brainkit: module %q is not registered", id)
	}
	decode, err := moduleConfigDecoder(cfg)
	if err != nil {
		return nil, fmt.Errorf("brainkit: module %q config: %w", id, err)
	}
	mod, err := factory.Build(bkmodule.BuildContext{
		FSRoot: fsRoot,
		Decode: decode,
	})
	if err != nil {
		return nil, fmt.Errorf("brainkit: build module %q: %w", id, err)
	}
	return mod, nil
}

func moduleConfigDecoder(cfg bkmodule.ModuleBuildConfig) (func(any) error, error) {
	var raw []byte
	switch {
	case cfg.YAML != "":
		raw = []byte(cfg.YAML)
	case len(bytes.TrimSpace(cfg.JSON)) > 0 && !bytes.Equal(bytes.TrimSpace(cfg.JSON), []byte("null")):
		if !json.Valid(cfg.JSON) {
			return nil, fmt.Errorf("json config is invalid")
		}
		raw = cfg.JSON
	default:
		return func(any) error { return nil }, nil
	}

	var node yaml.Node
	if err := yaml.Unmarshal(raw, &node); err != nil {
		return nil, err
	}
	return func(v any) error {
		if node.Kind == 0 {
			return nil
		}
		return node.Decode(v)
	}, nil
}
