package discovery

import (
	"testing"
	"time"
)

func TestStatic_ResolveAndBrowse(t *testing.T) {
	d := NewStatic(map[string]string{
		"server-1": "10.0.1.1:9090",
		"server-2": "10.0.1.2:9090",
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
	d := NewStatic(nil)
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

func TestMulticast_AnnounceAndDiscover(t *testing.T) {
	d1, err := NewMulticast("_test._tcp")
	if err != nil {
		t.Skipf("multicast not available: %v", err)
	}
	defer d1.Close()

	d2, err := NewMulticast("_test._tcp")
	if err != nil {
		t.Skipf("multicast not available: %v", err)
	}
	defer d2.Close()

	d1.Register(Peer{Name: "kit-1", Address: "127.0.0.1:9001"})

	time.Sleep(3 * time.Second)

	addr, err := d2.Resolve("kit-1")
	if err != nil {
		t.Fatalf("d2 did not discover kit-1: %v", err)
	}
	if addr != "127.0.0.1:9001" {
		t.Errorf("addr = %q", addr)
	}
}
