package registry

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/registry/registrymsg"
)

type mutationCall struct {
	op     string
	ctx    context.Context
	name   string
	typ    string
	config string
}

type fakeRegistryMutations struct {
	calls []mutationCall
	err   error
}

func (f *fakeRegistryMutations) AddRegistryProvider(ctx context.Context, name, typ string, raw json.RawMessage) error {
	f.calls = append(f.calls, mutationCall{op: "add_provider", ctx: ctx, name: name, typ: typ, config: string(raw)})
	return f.err
}

func (f *fakeRegistryMutations) RemoveRegistryProvider(ctx context.Context, name string) error {
	f.calls = append(f.calls, mutationCall{op: "remove_provider", ctx: ctx, name: name})
	return f.err
}

func (f *fakeRegistryMutations) AddRegistryStorage(ctx context.Context, name, typ string, raw json.RawMessage) error {
	f.calls = append(f.calls, mutationCall{op: "add_storage", ctx: ctx, name: name, typ: typ, config: string(raw)})
	return f.err
}

func (f *fakeRegistryMutations) RemoveRegistryStorage(ctx context.Context, name string) error {
	f.calls = append(f.calls, mutationCall{op: "remove_storage", ctx: ctx, name: name})
	return f.err
}

func (f *fakeRegistryMutations) AddRegistryVector(ctx context.Context, name, typ string, raw json.RawMessage) error {
	f.calls = append(f.calls, mutationCall{op: "add_vector", ctx: ctx, name: name, typ: typ, config: string(raw)})
	return f.err
}

func (f *fakeRegistryMutations) RemoveRegistryVector(ctx context.Context, name string) error {
	f.calls = append(f.calls, mutationCall{op: "remove_vector", ctx: ctx, name: name})
	return f.err
}

var _ bkmodule.RegistryMutationManager = (*fakeRegistryMutations)(nil)

func TestAddStorageDelegatesToRegistryMutationManager(t *testing.T) {
	mutations := &fakeRegistryMutations{}
	module := &Module{mutations: mutations}
	ctx := context.WithValue(context.Background(), struct{}{}, "add-storage")

	resp, err := module.AddStorage(ctx, registrymsg.StorageAddMsg{
		Name:   "docs",
		Type:   "memory",
		Config: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("add storage: %v", err)
	}
	if !resp.Added {
		t.Fatal("expected Added=true")
	}
	if len(mutations.calls) != 1 {
		t.Fatalf("mutation calls = %#v, want one call", mutations.calls)
	}
	got := mutations.calls[0]
	if got.op != "add_storage" || got.ctx != ctx || got.name != "docs" || got.typ != "memory" || got.config != "{}" {
		t.Fatalf("mutation call = %#v", got)
	}
}

func TestRemoveStorageDelegatesToRegistryMutationManager(t *testing.T) {
	mutations := &fakeRegistryMutations{}
	module := &Module{mutations: mutations}
	ctx := context.WithValue(context.Background(), struct{}{}, "remove-storage")

	resp, err := module.RemoveStorage(ctx, registrymsg.StorageRemoveMsg{Name: "docs"})
	if err != nil {
		t.Fatalf("remove storage: %v", err)
	}
	if !resp.Removed {
		t.Fatal("expected Removed=true")
	}
	if len(mutations.calls) != 1 {
		t.Fatalf("mutation calls = %#v, want one call", mutations.calls)
	}
	got := mutations.calls[0]
	if got.op != "remove_storage" || got.ctx != ctx || got.name != "docs" {
		t.Fatalf("mutation call = %#v", got)
	}
}

func TestAddVectorDelegatesToRegistryMutationManager(t *testing.T) {
	mutations := &fakeRegistryMutations{}
	module := &Module{mutations: mutations}
	raw := json.RawMessage(`{
			"path": "/tmp/docs.db",
			"connectionString": "postgres://unused",
			"uri": "mongodb://unused",
			"dbName": "vectors"
		}`)

	resp, err := module.AddVector(context.Background(), registrymsg.VectorAddMsg{
		Name:   "docs",
		Type:   "sqlite",
		Config: raw,
	})
	if err != nil {
		t.Fatalf("add vector: %v", err)
	}
	if !resp.Added {
		t.Fatal("expected Added=true")
	}
	if len(mutations.calls) != 1 {
		t.Fatalf("mutation calls = %#v, want one call", mutations.calls)
	}
	got := mutations.calls[0]
	if got.op != "add_vector" || got.name != "docs" || got.typ != "sqlite" || got.config != string(raw) {
		t.Fatalf("mutation call = %#v", got)
	}
}

func TestRemoveVectorDelegatesToRegistryMutationManager(t *testing.T) {
	mutations := &fakeRegistryMutations{}
	module := &Module{mutations: mutations}
	ctx := context.WithValue(context.Background(), struct{}{}, "remove-vector")

	resp, err := module.RemoveVector(ctx, registrymsg.VectorRemoveMsg{Name: "docs"})
	if err != nil {
		t.Fatalf("remove vector: %v", err)
	}
	if !resp.Removed {
		t.Fatal("expected Removed=true")
	}
	if len(mutations.calls) != 1 {
		t.Fatalf("mutation calls = %#v, want one call", mutations.calls)
	}
	got := mutations.calls[0]
	if got.op != "remove_vector" || got.ctx != ctx || got.name != "docs" {
		t.Fatalf("mutation call = %#v", got)
	}
}

func TestProviderAddReturnsRegistryMutationError(t *testing.T) {
	want := errors.New("runtime cache unavailable")
	mutations := &fakeRegistryMutations{err: want}
	module := &Module{mutations: mutations}

	_, err := module.AddProvider(context.Background(), registrymsg.ProviderAddMsg{
		Name:   "openai",
		Type:   "openai",
		Config: json.RawMessage(`{"APIKey":"test-key"}`),
	})
	if !errors.Is(err, want) {
		t.Fatalf("add provider error = %v, want %v", err, want)
	}
	if len(mutations.calls) != 1 {
		t.Fatalf("mutation calls = %#v, want one call", mutations.calls)
	}
	got := mutations.calls[0]
	if got.op != "add_provider" || got.name != "openai" || got.typ != "openai" || got.config != `{"APIKey":"test-key"}` {
		t.Fatalf("mutation call = %#v", got)
	}
}

func TestRemoveProviderDelegatesToRegistryMutationManager(t *testing.T) {
	mutations := &fakeRegistryMutations{}
	module := &Module{mutations: mutations}

	resp, err := module.RemoveProvider(context.Background(), registrymsg.ProviderRemoveMsg{Name: "openai"})
	if err != nil {
		t.Fatalf("remove provider: %v", err)
	}
	if !resp.Removed {
		t.Fatal("expected Removed=true")
	}
	if len(mutations.calls) != 1 {
		t.Fatalf("mutation calls = %#v, want one call", mutations.calls)
	}
	got := mutations.calls[0]
	if got.op != "remove_provider" || got.name != "openai" {
		t.Fatalf("mutation call = %#v", got)
	}
}

func TestFactoryDescriptorRequiresRegistryMutationManager(t *testing.T) {
	desc := bkmodule.NormalizeDescriptor("registry", Factory{}.Describe())
	var foundMutation bool
	for _, cap := range desc.Capabilities {
		switch cap.Name {
		case bkmodule.CapabilityRegistryMutation:
			foundMutation = true
			if cap.Direction != bkmodule.CapabilityRequired {
				t.Fatalf("registry mutation capability direction = %s, want required", cap.Direction)
			}
			if cap.Type == "" {
				t.Fatal("registry mutation capability should advertise a typed contract")
			}
		case "brainkit.core.storage_manager", "brainkit.core.registry_runtime_cache":
			t.Fatalf("registry module should not consume low-level mutation helper capability %s", cap.Name)
		}
	}
	if !foundMutation {
		t.Fatal("registry mutation capability descriptor missing")
	}
}
