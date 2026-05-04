package discovery

import (
	"context"
	"errors"
	"testing"
)

func TestStatic_ResolveAndBrowse(t *testing.T) {
	d := NewStaticFromConfig([]PeerConfig{
		{Name: "server-1", Address: "10.0.1.1:9090"},
		{Name: "server-2", Address: "10.0.1.2:9090"},
	})
	defer d.Close()

	addr, err := d.Resolve("server-1")
	if err != nil {
		t.Fatal(err)
	}
	if addr != "10.0.1.1:9090" {
		t.Errorf("resolve = %q", addr)
	}

	_, err = d.Resolve("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent peer")
	}

	peers, _ := d.Browse()
	if len(peers) != 2 {
		t.Errorf("browse = %d peers, want 2", len(peers))
	}
}

func TestStatic_Register(t *testing.T) {
	d := NewStaticFromConfig(nil)
	defer d.Close()

	d.Register(Peer{Name: "new-peer", Address: "10.0.1.3:9090"})

	addr, err := d.Resolve("new-peer")
	if err != nil {
		t.Fatal(err)
	}
	if addr != "10.0.1.3:9090" {
		t.Errorf("resolve = %q", addr)
	}
}

func TestStatic_BrowseNamespaces(t *testing.T) {
	d := NewStaticFromConfig([]PeerConfig{
		{Name: "a1", Namespace: "agents"},
		{Name: "a2", Namespace: "agents"},
		{Name: "g1", Namespace: "gateway"},
	})
	defer d.Close()

	namespaces, err := d.BrowseNamespaces()
	if err != nil {
		t.Fatal(err)
	}
	if len(namespaces) != 2 {
		t.Errorf("BrowseNamespaces = %d, want 2", len(namespaces))
	}
}

type failingProvider struct {
	err    error
	closes int
}

func (p *failingProvider) Resolve(string) (string, error) { return "", nil }
func (p *failingProvider) Browse() ([]Peer, error)        { return nil, nil }
func (p *failingProvider) BrowseNamespaces() ([]string, error) {
	return nil, nil
}
func (p *failingProvider) Register(Peer) error { return nil }
func (p *failingProvider) Close() error {
	p.closes++
	return p.err
}

func TestModuleCloseContextRetainsProviderOnCloseFailure(t *testing.T) {
	want := errors.New("close provider")
	provider := &failingProvider{err: want}
	module := &Module{provider: provider}

	if err := module.CloseContext(context.Background()); !errors.Is(err, want) {
		t.Fatalf("CloseContext error = %v, want %v", err, want)
	}
	if module.provider == nil {
		t.Fatal("provider was detached after close failure")
	}

	provider.err = nil
	if err := module.CloseContext(context.Background()); err != nil {
		t.Fatalf("retry CloseContext: %v", err)
	}
	if module.provider != nil {
		t.Fatal("provider remained attached after successful retry")
	}
	if provider.closes != 2 {
		t.Fatalf("provider closes = %d, want 2", provider.closes)
	}
}
