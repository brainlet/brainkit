package brainkit

import (
	"encoding/json"
	"errors"
	"testing"
)

// fakeFactory is a deterministic ModuleFactory that surfaces its
// decoded config so the test can assert the decode callback actually
// ran against the supplied payload.
type fakeFactory struct {
	desc ModuleDescriptor
	// built captures the last config the factory decoded — tests
	// inspect it to confirm the decode path.
	built *fakeConfig
}

type fakeConfig struct {
	Path    string `json:"path" yaml:"path"`
	Verbose bool   `json:"verbose" yaml:"verbose"`
}

func (f *fakeFactory) Build(ctx ModuleContext) (Module, error) {
	var cfg fakeConfig
	if err := ctx.Decode(&cfg); err != nil {
		return nil, err
	}
	f.built = &cfg
	return nil, nil
}

func (f *fakeFactory) Describe() ModuleDescriptor { return f.desc }

// jsonDecoder returns a decode func that unmarshals the supplied
// JSON blob. Cheap stand-in for yaml.Node.Decode in tests.
func jsonDecoder(raw string) func(any) error {
	return func(v any) error {
		if raw == "" {
			return nil
		}
		return json.Unmarshal([]byte(raw), v)
	}
}

func TestRegisterAndLookup(t *testing.T) {
	defer unregisterModuleForTest("fake")

	factory := &fakeFactory{desc: ModuleDescriptor{Name: "fake", Status: ModuleStatusBeta, Summary: "test-only"}}
	RegisterModule("fake", factory)

	got, ok := LookupModuleFactory("fake")
	if !ok {
		t.Fatal("LookupModuleFactory: want true, got false")
	}
	if got != factory {
		t.Fatalf("LookupModuleFactory: want %p, got %p", factory, got)
	}
}

func TestDoubleRegistrationPanics(t *testing.T) {
	defer unregisterModuleForTest("dup")

	RegisterModule("dup", &fakeFactory{})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("RegisterModule: expected panic on duplicate, got none")
		}
	}()
	RegisterModule("dup", &fakeFactory{})
}

func TestRegisterNilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("RegisterModule(nil): expected panic, got none")
		}
	}()
	RegisterModule("nilled", nil)
}

func TestRegisterEmptyNamePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("RegisterModule(\"\"): expected panic, got none")
		}
	}()
	RegisterModule("", &fakeFactory{})
}

func TestLookupMissing(t *testing.T) {
	if _, ok := LookupModuleFactory("definitely-not-registered"); ok {
		t.Fatal("LookupModuleFactory: want false for unknown, got true")
	}
}

func TestRegisteredNamesSorted(t *testing.T) {
	defer unregisterModuleForTest("aaa")
	defer unregisterModuleForTest("bbb")
	defer unregisterModuleForTest("ccc")

	RegisterModule("ccc", &fakeFactory{})
	RegisterModule("aaa", &fakeFactory{})
	RegisterModule("bbb", &fakeFactory{})

	names := RegisteredModuleNames()
	// Filter to the names this test owns — other tests + real
	// built-ins may coexist in the registry when running the full
	// package.
	var seen []string
	for _, n := range names {
		if n == "aaa" || n == "bbb" || n == "ccc" {
			seen = append(seen, n)
		}
	}
	want := []string{"aaa", "bbb", "ccc"}
	if len(seen) != len(want) {
		t.Fatalf("RegisteredModuleNames: want %v, got %v", want, seen)
	}
	for i, n := range want {
		if seen[i] != n {
			t.Fatalf("RegisteredModuleNames[%d]: want %q, got %q", i, n, seen[i])
		}
	}
}

func TestRegisteredModulesIncludesDescriptor(t *testing.T) {
	defer unregisterModuleForTest("described")
	defer unregisterModuleForTest("bare")

	RegisterModule("described", &fakeFactory{desc: ModuleDescriptor{
		Name: "described", Status: ModuleStatusStable, Summary: "has metadata",
	}})
	// A factory that doesn't implement ModuleDescriber still shows
	// up — just with a bare descriptor.
	RegisterModule("bare", ModuleFactoryFunc(func(ctx ModuleContext) (Module, error) {
		return nil, nil
	}))

	descs := RegisteredModules()

	var described, bare *ModuleDescriptor
	for i := range descs {
		switch descs[i].Name {
		case "described":
			described = &descs[i]
		case "bare":
			bare = &descs[i]
		}
	}
	if described == nil {
		t.Fatal("RegisteredModules: missing 'described'")
	}
	if described.Status != ModuleStatusStable {
		t.Fatalf("described.Status: want stable, got %q", described.Status)
	}
	if described.Summary != "has metadata" {
		t.Fatalf("described.Summary: want 'has metadata', got %q", described.Summary)
	}
	if bare == nil {
		t.Fatal("RegisteredModules: missing 'bare'")
	}
	if bare.Status != "" {
		t.Fatalf("bare.Status: want empty (no Describe impl), got %q", bare.Status)
	}
}

func TestFactoryDecodePath(t *testing.T) {
	defer unregisterModuleForTest("decodepath")

	factory := &fakeFactory{}
	RegisterModule("decodepath", factory)

	got, ok := LookupModuleFactory("decodepath")
	if !ok {
		t.Fatal("LookupModuleFactory failed")
	}

	_, err := got.Build(ModuleContext{FSRoot: "/tmp/brainkit", Decode: jsonDecoder(`{"path":"/tmp/x","verbose":true}`)})
	if err != nil {
		t.Fatalf("Build: unexpected err: %v", err)
	}
	if factory.built == nil {
		t.Fatal("Build: factory didn't capture config")
	}
	if factory.built.Path != "/tmp/x" {
		t.Fatalf("Build: Path=%q, want /tmp/x", factory.built.Path)
	}
	if !factory.built.Verbose {
		t.Fatal("Build: Verbose=false, want true")
	}
}

func TestFactoryFuncAdapter(t *testing.T) {
	defer unregisterModuleForTest("funced")

	var called bool
	RegisterModule("funced", ModuleFactoryFunc(func(ctx ModuleContext) (Module, error) {
		called = true
		var cfg fakeConfig
		if err := ctx.Decode(&cfg); err != nil {
			return nil, err
		}
		if cfg.Path != "/adapter" {
			return nil, errors.New("decode mismatch")
		}
		return nil, nil
	}))

	f, _ := LookupModuleFactory("funced")
	if _, err := f.Build(ModuleContext{Decode: jsonDecoder(`{"path":"/adapter"}`)}); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !called {
		t.Fatal("ModuleFactoryFunc: Build did not invoke underlying func")
	}
}
