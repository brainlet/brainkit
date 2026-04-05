package fixtures

import (
	"strings"
)

// FixtureNeeds describes what infrastructure a fixture requires.
type FixtureNeeds struct {
	Container    string // "postgres", "mongodb", "" for none
	Credential   string // "UPSTASH_REDIS_REST_URL", "" for none
	LibSQLServer bool   // needs libsql-server container (vector extensions)
	AI           bool   // needs OPENAI_API_KEY
	MCP          bool   // needs in-process MCP server
}

// containerBackends maps path segments to container names.
// A fixture needs a container if any segment in its relative path matches.
var containerBackends = map[string]string{
	"postgres":       "postgres",
	"postgres-scram": "postgres",
	"pgvector":       "postgres",
	"mongodb":        "mongodb",
	"mongodb-scram":  "mongodb",
}

// credentialBackends need cloud credentials, not containers.
var credentialBackends = map[string]string{
	"upstash": "UPSTASH_REDIS_REST_URL",
}

// vectorServerBackends need a libsql-server container (for vector extensions).
// "libsql" under vector/ needs the server; under memory/storage/ it's in-process.
var vectorServerBackends = map[string]bool{
	"libsql": true,
}

// aiCategories are top-level categories where every fixture needs AI.
var aiCategories = map[string]bool{
	"agent":        true,
	"ai":           true,
	"observability": true,
	"composition":  true,
}

// aiSegments trigger AI need when found anywhere in the path.
var aiSegments = map[string]bool{
	"with-agent-step":  true,
	"vector-query-tool": true,
	"with-llm-judge":   true,
	"semantic-recall":  true,
	"generate-title":   true,
	"working-memory":   true,
}

// ClassifyFixture determines what infrastructure a fixture needs based on its
// relative path from the fixtures root (e.g. "agent/generate/basic",
// "memory/storage/postgres", "vector/create-upsert-query/pgvector").
//
// Classification scans ALL path segments, so nesting depth doesn't matter.
func ClassifyFixture(relPath string) FixtureNeeds {
	var needs FixtureNeeds

	segments := strings.Split(relPath, "/")
	if len(segments) < 2 {
		return needs
	}

	category := segments[0]

	// MCP detection
	if category == "mcp" {
		needs.MCP = true
	}

	// AI detection: category-level
	if aiCategories[category] {
		needs.AI = true
	}

	// Scan all segments for infrastructure needs
	for _, seg := range segments {
		// Container detection
		if backend, ok := containerBackends[seg]; ok && needs.Container == "" {
			needs.Container = backend
		}

		// Credential detection
		if envVar, ok := credentialBackends[seg]; ok && needs.Credential == "" {
			needs.Credential = envVar
		}

		// LibSQL server detection: only under vector/ category
		// (memory/storage/libsql is in-process, no container needed)
		if vectorServerBackends[seg] && category == "vector" {
			needs.LibSQLServer = true
		}

		// AI segment detection
		if aiSegments[seg] {
			needs.AI = true
		}
	}

	// memory/storage/* all need AI (they use Agent conversation)
	if category == "memory" && hasSegment(segments, "storage") {
		needs.AI = true
	}

	return needs
}

// hasSegment checks if a specific segment exists in the path.
func hasSegment(segments []string, target string) bool {
	for _, seg := range segments {
		if seg == target {
			return true
		}
	}
	return false
}
