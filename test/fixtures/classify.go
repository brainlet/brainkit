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

// containerBackends maps fixture path segments to the container they need.
// A fixture needs a container if any segment in its relative path matches.
var containerBackends = map[string]string{
	"postgres-basic":          "postgres",
	"postgres-scram":          "postgres",
	"mongodb-basic":           "mongodb",
	"mongodb-scram":           "mongodb",
	"with-memory-postgres":    "postgres",
	"with-memory-mongodb":     "mongodb",
	"semantic-recall":         "postgres", // needs pgvector + libsql-server
	"working-memory":          "postgres", // needs pgvector + libsql-server
	"vector-query-tool":       "postgres", // needs pgvector + libsql-server
	"pgvector-create-upsert-query": "postgres",
	"pgvector-methods":        "postgres",
	"mongodb-create-upsert-query":  "mongodb",
	"mongodb-methods":         "mongodb",
	"libsql-create-upsert-query":   "",  // libsql-server container, handled by LibSQLServer flag
	"libsql-methods":          "",        // libsql-server container, handled by LibSQLServer flag
}

// credentialBackends maps fixture path segments to the env var they need.
var credentialBackends = map[string]string{
	"upstash-basic":       "UPSTASH_REDIS_REST_URL",
	"with-memory-upstash": "UPSTASH_REDIS_REST_URL",
}

// vectorServerBackends maps fixture name segments that need a libsql-server container.
var vectorServerBackends = map[string]bool{
	"semantic-recall":            true,
	"working-memory":             true,
	"vector-query-tool":          true,
	"libsql-create-upsert-query": true,
	"libsql-methods":             true,
}

// aiCategories are top-level categories where every fixture needs AI.
var aiCategories = map[string]bool{
	"agent":        true,
	"ai":           true,
	"observability": true,
	"composition":  true,
}

// aiSegments are specific fixture names that need AI regardless of category.
var aiSegments = map[string]bool{
	"with-agent-step":  true,
	"vector-query-tool": true,
	// memory fixtures that use Agent conversation
	"inmemory-basic":  true,
	"libsql-basic":    true,
	"libsql-local":    true,
	"postgres-basic":  true,
	"postgres-scram":  true,
	"mongodb-basic":   true,
	"mongodb-scram":   true,
	"upstash-basic":   true,
	"semantic-recall":  true,
	"working-memory":  true,
}

// vectorCategory marks categories where ALL fixtures need containers.
var vectorCategory = map[string]bool{
	"vector": true,
}

// ClassifyFixture determines what infrastructure a fixture needs based on its
// relative path from the fixtures root (e.g. "agent/generate-basic").
func ClassifyFixture(relPath string) FixtureNeeds {
	var needs FixtureNeeds

	parts := strings.Split(relPath, "/")
	if len(parts) < 2 {
		return needs
	}

	category := parts[0]
	name := parts[len(parts)-1] // leaf segment (fixture name)

	// MCP detection: category == "mcp"
	if category == "mcp" {
		needs.MCP = true
	}

	// AI detection: full category match or specific fixture name
	if aiCategories[category] {
		needs.AI = true
	}
	if aiSegments[name] && category == "memory" {
		needs.AI = true
	}
	if aiSegments[name] && category == "workflow" {
		needs.AI = true
	}
	if aiSegments[name] && category == "rag" {
		needs.AI = true
	}

	// Container detection: category-level or name-level
	if vectorCategory[category] {
		// All vector fixtures need a container
		if strings.HasPrefix(name, "pgvector") {
			needs.Container = "postgres"
		} else if strings.HasPrefix(name, "mongodb") {
			needs.Container = "mongodb"
		}
		// libsql-* vector fixtures need libsql-server but not postgres/mongodb container
	}
	if c, ok := containerBackends[name]; ok && needs.Container == "" {
		needs.Container = c
	}

	// LibSQL server detection (vector extensions)
	if vectorServerBackends[name] {
		needs.LibSQLServer = true
	}
	// All libsql-* fixtures in vector category need the server
	if category == "vector" && strings.HasPrefix(name, "libsql") {
		needs.LibSQLServer = true
	}

	// Credential detection
	if cred, ok := credentialBackends[name]; ok {
		needs.Credential = cred
	}
	// Also check for "upstash" anywhere in name
	if strings.Contains(name, "upstash") && needs.Credential == "" {
		needs.Credential = "UPSTASH_REDIS_REST_URL"
	}

	return needs
}
