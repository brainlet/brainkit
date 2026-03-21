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
