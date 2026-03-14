package brainkit

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
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { kit.Close() })
	return kit
}

// newTestKitNoKey creates a Kit without API keys — for bus/registry tests.
func newTestKitNoKey(t *testing.T) *Kit {
	t.Helper()
	kit, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { kit.Close() })
	return kit
}
