package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	bkmodule "github.com/brainlet/brainkit/module"
)

type lifecycleDebugRegistry struct {
	mu        sync.Mutex
	nextID    uint64
	providers map[string]lifecycleDebugProvider
}

type lifecycleDebugProvider struct {
	id       uint64
	snapshot func() any
}

func newLifecycleDebugRegistry() *lifecycleDebugRegistry {
	return &lifecycleDebugRegistry{providers: map[string]lifecycleDebugProvider{}}
}

func (r *lifecycleDebugRegistry) RegisterLifecycleDebug(ctx context.Context, name string, snapshot func() any) (bkmodule.Handle, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("lifecycle debug component name is required")
	}
	if snapshot == nil {
		return nil, fmt.Errorf("lifecycle debug component %q requires a snapshot function", name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[name]; exists {
		return nil, fmt.Errorf("lifecycle debug component %q already registered", name)
	}
	r.nextID++
	id := r.nextID
	r.providers[name] = lifecycleDebugProvider{id: id, snapshot: snapshot}
	return bkmodule.HandleFunc(func(context.Context) error {
		r.mu.Lock()
		defer r.mu.Unlock()
		current, ok := r.providers[name]
		if ok && current.id == id {
			delete(r.providers, name)
		}
		return nil
	}), nil
}

func (r *lifecycleDebugRegistry) snapshot() []bkmodule.LifecycleDebugComponent {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	names := make([]string, 0, len(r.providers))
	providers := make(map[string]lifecycleDebugProvider, len(r.providers))
	for name, provider := range r.providers {
		names = append(names, name)
		providers[name] = provider
	}
	r.mu.Unlock()
	sort.Strings(names)

	out := make([]bkmodule.LifecycleDebugComponent, 0, len(names))
	for _, name := range names {
		out = append(out, snapshotLifecycleDebugComponent(name, providers[name].snapshot))
	}
	return out
}

func snapshotLifecycleDebugComponent(name string, snapshot func() any) (component bkmodule.LifecycleDebugComponent) {
	component.Name = name
	defer func() {
		if recovered := recover(); recovered != nil {
			component.Data = nil
			component.Error = fmt.Sprintf("panic: %v", recovered)
		}
	}()
	data, err := json.Marshal(snapshot())
	if err != nil {
		component.Error = err.Error()
		return component
	}
	if len(data) == 0 {
		data = []byte("null")
	}
	component.Data = json.RawMessage(data)
	return component
}

func (k *Kit) lifecycleDebugSnapshot() bkmodule.LifecycleDebugSnapshot {
	runtime := k.kernel.LifecycleDebugSnapshot()
	runtime.MountedModules = len(k.MountedModules())
	return bkmodule.LifecycleDebugSnapshot{
		Runtime:    runtime,
		Components: k.lifecycleDebug.snapshot(),
	}
}
