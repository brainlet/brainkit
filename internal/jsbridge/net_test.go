package jsbridge

import (
	"net"
	"strconv"
	"testing"
)

func TestNetJoinHostPort_IPv4(t *testing.T) {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(5432))
	if addr != "127.0.0.1:5432" {
		t.Errorf("IPv4: got %q, want %q", addr, "127.0.0.1:5432")
	}
}

func TestNetJoinHostPort_IPv6(t *testing.T) {
	addr := net.JoinHostPort("::1", strconv.Itoa(5432))
	if addr != "[::1]:5432" {
		t.Errorf("IPv6 loopback: got %q, want %q", addr, "[::1]:5432")
	}

	addr2 := net.JoinHostPort("2001:db8::1", strconv.Itoa(27017))
	if addr2 != "[2001:db8::1]:27017" {
		t.Errorf("IPv6 full: got %q, want %q", addr2, "[2001:db8::1]:27017")
	}
}

func TestNetJoinHostPort_Hostname(t *testing.T) {
	addr := net.JoinHostPort("db.example.com", strconv.Itoa(5432))
	if addr != "db.example.com:5432" {
		t.Errorf("hostname: got %q, want %q", addr, "db.example.com:5432")
	}
}

func TestSocket_ExtendsDuplex(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Events(), NodeStreams(), Timers(), Net())
	result := evalString(t, b, `
		var s = new globalThis.__node_net.Socket();
		JSON.stringify({
			isReadable: typeof s.pipe === "function",
			isDuplex: typeof s.write === "function" && typeof s.push === "function",
			hasAsyncIterator: typeof s[Symbol.asyncIterator] === "function",
			hasConnect: typeof s.connect === "function",
			hasSetNoDelay: typeof s.setNoDelay === "function",
		});
	`)
	expected := `{"isReadable":true,"isDuplex":true,"hasAsyncIterator":true,"hasConnect":true,"hasSetNoDelay":true}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}

func TestSocket_CreateConnection(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Events(), NodeStreams(), Timers(), Net())
	result := evalString(t, b, `
		var N = globalThis.__node_net;
		JSON.stringify({
			hasSocket: typeof N.Socket === "function",
			hasCreateConnection: typeof N.createConnection === "function",
			hasConnect: typeof N.connect === "function",
			hasIsIP: typeof N.isIP === "function",
		});
	`)
	expected := `{"hasSocket":true,"hasCreateConnection":true,"hasConnect":true,"hasIsIP":true}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}
