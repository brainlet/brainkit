package jsbridge

import (
	"fmt"
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
		var s = new globalThis.net.Socket();
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
		var N = globalThis.net;
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

func TestNetResourceSnapshotTracksOpenSocket(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := ln.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	t.Cleanup(func() {
		select {
		case conn := <-accepted:
			_ = conn.Close()
		default:
		}
	})

	b := newTestBridge(t, Console(), Encoding(), Events(), NodeStreams(), Timers(), Net())
	val, err := b.EvalAsync("net-resource.js", fmt.Sprintf(`(async () => {
		return await new Promise((resolve, reject) => {
			const socket = net.createConnection({ host: "127.0.0.1", port: %d });
			globalThis.__resourceSocket = socket;
			socket.on("connect", () => resolve("connected"));
			socket.on("error", reject);
		});
	})()`, ln.Addr().(*net.TCPAddr).Port))
	if err != nil {
		t.Fatalf("EvalAsync connect: %v", err)
	}
	val.Free()

	if got := resourceCount(b, "net.tcpSockets"); got != 1 {
		t.Fatalf("net socket resources = %d, want 1 snapshot=%+v", got, b.DebugSnapshot())
	}
}

func TestTLSModuleShape(t *testing.T) {
	b := newTestBridge(t, Console(), Encoding(), Events(), NodeStreams(), Timers(), Net())
	result := evalString(t, b, `
		var threw = false;
		try { globalThis.tls.connect({}); } catch (_) { threw = true; }
		var netBoundary = {};
		try { globalThis.net.createServer(); } catch (err) {
			netBoundary = { code: err.code || "", boundaryClass: err.boundaryClass || "", api: err.api || "" };
		}
		var tlsBoundary = {};
		try { globalThis.tls.createServer(); } catch (err) {
			tlsBoundary = { code: err.code || "", boundaryClass: err.boundaryClass || "", api: err.api || "" };
		}
		JSON.stringify({
			hasConnect: typeof globalThis.tls.connect === "function",
			hasCreateServer: typeof globalThis.tls.createServer === "function",
			hasTLSSocket: typeof globalThis.tls.TLSSocket === "function",
			minVersion: globalThis.tls.DEFAULT_MIN_VERSION,
			throwsWithoutSocket: threw,
			netBoundary: netBoundary,
			tlsBoundary: tlsBoundary
		});
	`)
	expected := `{"hasConnect":true,"hasCreateServer":true,"hasTLSSocket":true,"minVersion":"TLSv1.2","throwsWithoutSocket":true,"netBoundary":{"code":"BRAINKIT_UNSUPPORTED_BOUNDARY","boundaryClass":"server-listener","api":"net.createServer"},"tlsBoundary":{"code":"BRAINKIT_UNSUPPORTED_BOUNDARY","boundaryClass":"server-listener","api":"tls.createServer"}}`
	if result != expected {
		t.Errorf("got %s", result)
	}
}
