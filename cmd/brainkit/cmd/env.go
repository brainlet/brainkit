package cmd

import (
	"bufio"
	"os"
	"strings"
)

// loadDotEnv reads key=value pairs from `./.env` (if it exists) and
// sets each entry in the process environment via `os.Setenv` —
// matching the convention `server.LoadConfig`'s `expandEnv` expects
// when the yaml references `$OPENAI_API_KEY` / `${VAR}`.
//
// Silent no-op when `.env` is absent. Existing env vars always win
// — we don't overwrite — so a `BRAINKIT_SECRET_KEY=...` on the
// command line stays authoritative.
//
// Called from `brainkit start` before the server config loads. The
// other CLI verbs don't need this because they forward requests to
// an already-running server that resolved its own env at boot time.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip `export ` prefix — common in shell-style .env files.
		line = strings.TrimPrefix(line, "export ")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Strip matching surrounding quotes (" or ').
		if len(val) >= 2 {
			first, last := val[0], val[len(val)-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if key == "" {
			continue
		}
		if _, set := os.LookupEnv(key); set {
			continue
		}
		_ = os.Setenv(key, val)
	}
}
