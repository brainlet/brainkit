package jsbridge

import (
	"fmt"
	"os"
	"runtime"
	"testing"
)

func TestOS_Platform(t *testing.T) {
	b := newTestBridge(t, Console(), OS())
	result := evalString(t, b, `globalThis.os.platform()`)
	if result != runtime.GOOS {
		t.Errorf("got %s, want %s", result, runtime.GOOS)
	}
}

func TestOS_Arch(t *testing.T) {
	b := newTestBridge(t, Console(), OS())
	result := evalString(t, b, `globalThis.os.arch()`)
	expected := runtime.GOARCH
	if expected == "amd64" {
		expected = "x64"
	} else if expected == "386" {
		expected = "ia32"
	}
	if result != expected {
		t.Errorf("got %s, want %s", result, expected)
	}
}

func TestOS_Tmpdir(t *testing.T) {
	b := newTestBridge(t, Console(), OS())
	result := evalString(t, b, `globalThis.os.tmpdir()`)
	if result != os.TempDir() {
		t.Errorf("got %s, want %s", result, os.TempDir())
	}
}

func TestOS_Homedir(t *testing.T) {
	b := newTestBridge(t, Console(), OS())
	result := evalString(t, b, `globalThis.os.homedir()`)
	expected, _ := os.UserHomeDir()
	if result != expected {
		t.Errorf("got %s, want %s", result, expected)
	}
}

func TestOS_Hostname(t *testing.T) {
	b := newTestBridge(t, Console(), OS())
	result := evalString(t, b, `globalThis.os.hostname()`)
	expected, _ := os.Hostname()
	if result != expected {
		t.Errorf("got %s, want %s", result, expected)
	}
}

func TestOS_Type(t *testing.T) {
	b := newTestBridge(t, Console(), OS())
	result := evalString(t, b, `globalThis.os.type()`)
	expected := runtime.GOOS
	switch expected {
	case "darwin":
		expected = "Darwin"
	case "linux":
		expected = "Linux"
	case "windows":
		expected = "Windows_NT"
	}
	if result != expected {
		t.Errorf("got %s, want %s", result, expected)
	}
}

func TestOS_EOL(t *testing.T) {
	b := newTestBridge(t, Console(), OS())
	result := evalString(t, b, `globalThis.os.EOL`)
	expected := "\n"
	if runtime.GOOS == "windows" {
		expected = "\r\n"
	}
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestOS_Cpus(t *testing.T) {
	b := newTestBridge(t, Console(), OS())
	result := evalString(t, b, `globalThis.os.cpus().length.toString()`)
	expected := runtime.NumCPU()
	if result != fmt.Sprintf("%d", expected) {
		t.Errorf("got %s, want %d", result, expected)
	}
}
