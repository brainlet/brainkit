package jsbridge

import (
	"testing"
)

func TestDNS_LookupLocalhost(t *testing.T) {
	b := newTestBridge(t, DNS())
	val, err := b.Eval("test.js", `
		var result = JSON.parse(__go_dns_lookup("localhost"));
		JSON.stringify({ addr: result.address, fam: result.family });
	`)
	if err != nil {
		t.Fatalf("dns.lookup localhost: %v", err)
	}
	defer val.Free()
	t.Logf("dns.lookup localhost: %s", val.String())
	// Should resolve to 127.0.0.1 or ::1
	s := val.String()
	if s == "" {
		t.Fatal("dns.lookup returned empty")
	}
}

func TestDNS_JSLookupCallback(t *testing.T) {
	b := newTestBridge(t, DNS())
	val, err := b.Eval("test.js", `
		var result = null;
		globalThis.dns.lookup("localhost", function(err, addr, family) {
			if (err) throw err;
			result = { addr: addr, family: family };
		});
		JSON.stringify(result);
	`)
	if err != nil {
		t.Fatalf("dns.lookup callback: %v", err)
	}
	defer val.Free()
	s := val.String()
	if s == "" || s == "null" {
		t.Fatal("dns.lookup callback returned null")
	}
	t.Logf("dns.lookup callback: %s", s)
}
