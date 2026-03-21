package kit

import (
	"os"
	"strings"
	"testing"
)

func loadEnv(t *testing.T) {
	t.Helper()
	for _, path := range []string{".env", "../.env"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if k, v, ok := strings.Cut(line, "="); ok {
				// Strip surrounding quotes from value
				if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
					v = v[1 : len(v)-1]
				}
				os.Setenv(k, v)
			}
		}
		return
	}
}

func requireKey(t *testing.T) string {
	t.Helper()
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	return key
}

func newTestKit(t *testing.T) *Kit {
	t.Helper()
	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"OPENAI_API_KEY": key,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { kit.Close() })
	return kit
}

func newTestKitNoKey(t *testing.T) *Kit {
	t.Helper()
	kit, err := New(Config{Namespace: "test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { kit.Close() })
	return kit
}

// loadFixture reads a test fixture file from testdata/.
func loadFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("load fixture %s: %v", path, err)
	}
	return string(data)
}
