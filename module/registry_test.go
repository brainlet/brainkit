package module

import (
	"encoding/json"
	"testing"
)

type fakeFactory struct {
	desc  Descriptor
	built *fakeConfig
}

type fakeConfig struct {
	Path    string `json:"path" yaml:"path"`
	Verbose bool   `json:"verbose" yaml:"verbose"`
}

func (f *fakeFactory) Build(ctx BuildContext) (Module, error) {
	var cfg fakeConfig
	if err := ctx.Decode(&cfg); err != nil {
		return nil, err
	}
	f.built = &cfg
	return nil, nil
}

func (f *fakeFactory) Describe() Descriptor { return f.desc }

func unregisterForTest(name string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(registry, name)
}

func jsonDecoder(raw string) func(any) error {
	return func(v any) error {
		if raw == "" {
			return nil
		}
		return json.Unmarshal([]byte(raw), v)
	}
}

func TestRegisterAndLookup(t *testing.T) {
	defer unregisterForTest("fake")

	factory := &fakeFactory{desc: Descriptor{Name: "fake", Status: StatusBeta, Summary: "test-only"}}
	Register("fake", factory)

	got, ok := Lookup("fake")
	if !ok {
		t.Fatal("Lookup: want true, got false")
	}
	if got != factory {
		t.Fatalf("Lookup: want %p, got %p", factory, got)
	}
}

func TestDoubleRegistrationPanics(t *testing.T) {
	defer unregisterForTest("dup")

	Register("dup", &fakeFactory{})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Register: expected panic on duplicate, got none")
		}
	}()
	Register("dup", &fakeFactory{})
}

func TestRegisterNilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Register(nil): expected panic, got none")
		}
	}()
	Register("nilled", nil)
}

func TestRegisterEmptyNamePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Register(\"\"): expected panic, got none")
		}
	}()
	Register("", &fakeFactory{})
}

func TestLookupMissing(t *testing.T) {
	if _, ok := Lookup("definitely-not-registered"); ok {
		t.Fatal("Lookup: want false for unknown, got true")
	}
}

func TestRegisteredNamesSorted(t *testing.T) {
	defer unregisterForTest("aaa")
	defer unregisterForTest("bbb")
	defer unregisterForTest("ccc")

	Register("ccc", &fakeFactory{})
	Register("aaa", &fakeFactory{})
	Register("bbb", &fakeFactory{})

	names := Names()
	var seen []string
	for _, n := range names {
		if n == "aaa" || n == "bbb" || n == "ccc" {
			seen = append(seen, n)
		}
	}
	want := []string{"aaa", "bbb", "ccc"}
	if len(seen) != len(want) {
		t.Fatalf("Names: want %v, got %v", want, seen)
	}
	for i, n := range want {
		if seen[i] != n {
			t.Fatalf("Names[%d]: want %q, got %q", i, n, seen[i])
		}
	}
}

func TestDescriptorsIncludesDescriptor(t *testing.T) {
	defer unregisterForTest("described")
	defer unregisterForTest("bare")

	Register("described", &fakeFactory{desc: Descriptor{
		Name: "described", Status: StatusStable, Summary: "has metadata",
	}})
	Register("bare", FactoryFunc(func(ctx BuildContext) (Module, error) {
		return nil, nil
	}))

	descs := Descriptors()

	var described, bare *Descriptor
	for i := range descs {
		switch descs[i].Name {
		case "described":
			described = &descs[i]
		case "bare":
			bare = &descs[i]
		}
	}
	if described == nil {
		t.Fatal("Descriptors: missing 'described'")
	}
	if described.Status != StatusStable {
		t.Fatalf("described.Status: want stable, got %q", described.Status)
	}
	if described.Summary != "has metadata" {
		t.Fatalf("described.Summary: want 'has metadata', got %q", described.Summary)
	}
	if bare == nil {
		t.Fatal("Descriptors: missing 'bare'")
	}
	if bare.Status != "" {
		t.Fatalf("bare.Status: want empty, got %q", bare.Status)
	}
}

func TestFactoryDecodePath(t *testing.T) {
	defer unregisterForTest("decodepath")

	factory := &fakeFactory{}
	Register("decodepath", factory)

	got, ok := Lookup("decodepath")
	if !ok {
		t.Fatal("Lookup failed")
	}

	_, err := got.Build(BuildContext{FSRoot: "/tmp/brainkit", Decode: jsonDecoder(`{"path":"/tmp/x","verbose":true}`)})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if factory.built == nil {
		t.Fatal("factory did not decode config")
	}
	if factory.built.Path != "/tmp/x" || !factory.built.Verbose {
		t.Fatalf("decoded config mismatch: %+v", *factory.built)
	}
}

func TestFactoryDecodeErrorPropagates(t *testing.T) {
	defer unregisterForTest("decodeerr")

	Register("decodeerr", &fakeFactory{})
	got, _ := Lookup("decodeerr")
	_, err := got.Build(BuildContext{Decode: jsonDecoder(`{"path":`)})
	if err == nil {
		t.Fatal("Build: expected decode error")
	}
}

func TestFactoryFunc(t *testing.T) {
	defer unregisterForTest("funced")

	called := false
	Register("funced", FactoryFunc(func(ctx BuildContext) (Module, error) {
		called = true
		return nil, nil
	}))

	f, _ := Lookup("funced")
	if _, err := f.Build(BuildContext{Decode: jsonDecoder(`{"path":"/adapter"}`)}); err != nil {
		t.Fatalf("FactoryFunc Build: %v", err)
	}
	if !called {
		t.Fatal("FactoryFunc: Build did not invoke underlying func")
	}
}
